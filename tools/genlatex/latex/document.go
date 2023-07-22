package latex

import (
	"context"
	"github.com/peter-mount/documentation/tools/genlatex/latex/util"
	"github.com/peter-mount/documentation/tools/genlatex/parser"
	"golang.org/x/net/html"
	"strings"
	"time"
)

func (c *Converter) beginDocument(n *html.Node, ctx context.Context) error {
	s := c.Stylesheet()

	_ = util.Writef(ctx, "%% Generated %s\n", time.Now().Format(time.RFC3339))

	_ = util.WriteString(ctx, `\documentclass[
    a4paper, % Page size
    fontsize=10pt, % Base font size
    twoside=true, % Use different layouts for even and odd pages (in particular, if twoside=true, the margin column will be always on the outside)
	%open=any, % If twoside=true, uncomment this to force new chapters to start on any page, not only on right (odd) pages
	%chapterentrydots=true, % Uncomment to output dots from the chapter name to the page number in the table of contents
	numbers=noenddot, % Comment to output dots after chapter numbers; the most common values for this option are: enddot, noenddot and auto (see the KOMAScript documentation for an in-depth explanation)
	fontmethod=tex, % Can also use "modern" with XeLaTeX or LuaTex; "tex" is the default for PdfLaTex, and "modern" is the default for those two.
]{kaobook}

% Choose the language
\ifxetexorluatex
	\usepackage{polyglossia}
	\setmainlanguage{english}
\else
	\usepackage[english]{babel} % Load characters and hyphenation
\fi
\usepackage[english=british]{csquotes}	% English quotes
`)

	for _, p := range s.UsePackage {
		_ = util.Writef(ctx, "\\usepackage{%s}\n", strings.TrimSpace(p))
	}

	_ = util.WriteString(ctx, `
\usepackage{area51}

% Load the bibliography package
\usepackage{kaobiblio}
\addbibresource{main.bib} % Bibliography file

% Load mathematical packages for theorems and related environments
\usepackage[framed=true]{kaotheorems}

% Load the package for hyperreferences
\usepackage{kaorefs}

\graphicspath{{examples/documentation/images/}{images/}} % Paths in which to look for images

\makeindex[columns=3, title=Alphabetical Index, intoc] % Make LaTeX produce the files required to compile the index

\makeglossaries % Make LaTeX produce the files required to compile the glossary
\input{glossary.tex} % Include the glossary definitions

\makenomenclature % Make LaTeX produce the files required to compile the nomenclature

% Reset sidenote counter at chapters
%\counterwithin*{sidenote}{chapter}

%----------------------------------------------------------------------------------------

`)

	for _, p := range s.Preamble {
		_ = util.WriteStringLn(ctx, p)
	}

	_ = util.WriteStringLn(ctx, `\begin{document}`)

	info := make(map[string]string)

	// Look for bookMeta "object"
	meta := parser.FindById(n, "bookMeta")
	if meta != nil {
		_ = parser.HandleChildren(func(n *html.Node, ctx context.Context) error {
			if n.Type == html.ElementNode {
				key := n.Data
				value := parser.GetText(n)
				switch key {

				// These append "*" and wrap with () but do not escape
				case "cover":
					key = key + "*"
					value = "{" + value + "}"

				// Don't escape the text or change key
				case "date", "copyright":
					value = "{" + value + "}"

				// Default wrap {} and escape the value
				default:
					value = "{" + EscapeText(value) + "}"
				}
				info[key] = value
			}
			return nil
		}, meta, ctx)
	}

	for _, k := range []string{"titlehead", "subject", "title", "subtitle", "author", "date", "publishers"} {
		if v, ok := info[k]; ok {
			_ = util.Writef(ctx, "\\%s%s\n", k, v)
		}
	}

	_ = util.WriteString(ctx, `\frontmatter

\makeatletter
\uppertitleback{\@titlehead} % Header

\lowertitleback{
`)

	if copyright, ok := info["copyright"]; ok {
		_ = util.Writef(ctx, `
	\textbf{Copyright}\\
	This book is copyright %s.
	\medskip
	
`, copyright)
	}

	_ = util.WriteString(ctx, `
	\textbf{Colophon} \\
	This document was typeset with the help of \href{https://www.latex-project.org/}{\LaTeX} using the \href{https://github.com/fmarotta/kaobook/}{kaobook} class.
	
	This document is available to download at: \\\url{https://area51.dev}

	The source code of this book is available at:\\\url{https://github.com/peter-mount/documentation}
	
	(You are welcome to contribute!)
	
	\medskip
	
	\textbf{Publisher} \\
	First printed in May 2019 by \@publishers
}
\makeatother
`)

	// Dedication page here

	_ = util.WriteString(ctx, `

% Note that \maketitle outputs the pages before here

\maketitle

%----------------------------------------------------------------------------------------
%	TABLE OF CONTENTS & LIST OF FIGURES/TABLES
%----------------------------------------------------------------------------------------

\begingroup % Local scope for the following commands

% Define the style for the TOC, LOF, and LOT
%\setstretch{1} % Uncomment to modify line spacing in the ToC
%\hypersetup{linkcolor=blue} % Uncomment to set the colour of links in the ToC
\setlength{\textheight}{230\hscale} % Manually adjust the height of the ToC pages

% Turn on compatibility mode for the etoc package
\etocstandarddisplaystyle % "toc display" as if etoc was not loaded
\etocstandardlines % "toc lines as if etoc was not loaded

\tableofcontents % Output the table of contents

\listoffigures % Output the list of figures

% Comment both of the following lines to have the LOF and the LOT on 
% different pages
\let\cleardoublepage\bigskip
\let\clearpage\bigskip

\listoftables % Output the list of tables

\listoflstlistings % Output the list of listings

\endgroup

%----------------------------------------------------------------------------------------
%	MAIN BODY
%----------------------------------------------------------------------------------------

\mainmatter % Denotes the start of the main document content, resets page numbering and uses arabic numbers
\setchapterstyle{kao} % Choose the default chapter heading style

`)
	return nil
}

func (c *Converter) endDocument(n *html.Node, ctx context.Context) error {

	_ = util.WriteString(ctx, `

%----------------------------------------------------------------------------------------

\backmatter % Denotes the end of the main document content
\setchapterstyle{plain} % Output plain chapters from this point onwards

%----------------------------------------------------------------------------------------
%	GLOSSARY
%----------------------------------------------------------------------------------------

% The glossary needs to be compiled on the command line with 'makeglossaries main' from the template directory

%\setglossarystyle{listgroup} % Set the style of the glossary (see https://en.wikibooks.org/wiki/LaTeX/Glossary for a reference)
%\printglossary[title=Special Terms, toctitle=List of Terms] % Output the glossary, 'title' is the chapter heading for the glossary, toctitle is the table of contents heading

%----------------------------------------------------------------------------------------
%	INDEX
%----------------------------------------------------------------------------------------

% The index needs to be compiled on the command line with 'makeindex main' from the template directory

\printindex % Output the index

%----------------------------------------------------------------------------------------
%	BACK COVER
%----------------------------------------------------------------------------------------

% If you have a PDF/image file that you want to use as a back cover, uncomment the following lines

%\clearpage
%\thispagestyle{empty}
%\null%
%\clearpage
%\includepdf{cover-back.pdf}

%----------------------------------------------------------------------------------------

\end{document}

`)

	return nil
}

# Report API

This is a simple API for generating reports in PDF format. It uses the [pdf](..) package to generate the PDF, but introduces an automatic flow layout where `ContentBlock`s are assigned to pages and rendered in the order they are added.

Besides a sequences of `ContentBlock`s, the report API also supports page-specific headers and footers.


## Usage overview

In typical usage, the application will initialize a `Theme` - an object that contains the default styles for the report. This doubles as a facade to the `pdf.Document` configuration and provides a set of configured block types. And header and footer blocks.

With an initialized theme, the application can generate PDF documents from a JSON input file, typically provided by the client. The JSON input provides basic metadata and a sequence of blocks that will be rendered in the order they are provided.

Block types may optionally implement page-breaking logic, or they will be added to the first page
with sufficient space.

`Report.KeepTogether` groups blocks so they move to the next page when the
group fits on a fresh page but not in the remaining space. A heading plus the
first body block, or a signature that must stay intact:

```go
r.KeepTogether(r.Text("<h2>Agenda</h2>"), firstParagraph)
r.KeepTogether(signature)
```

If the group is taller than a page, it splits like `Composite` so layout
cannot loop or drop content. Nested and adjacent groups are supported.
A `ContentBlock` may also implement `KeepTogetherHint() bool` to opt into the
same page-break behavior without wrapping.

Keep-together is honored for the current flow block and for groups nested in
`Composite`, `Div`, and the other built-in containers.


## Block content

Blocks are simple objects that implement the `ContentBlock` interface. They are responsible for measuring their content and drawing themselves on the canvas. Many block types support nested blocks, which will be measured and drawn in the order they are added.

We aim to support the building of complex block types from simpler ones. Often, implementing block types is a matter of composing simpler blocks and providing configuration options. As an example, a table can be mostly implemented by recursively adding `Div` blocks with the appropriate padding and background, and ensuring that necessary text styles are available.


## Notes

[NOTES.md](NOTES.md) tracks nonobvious properties of the module and past layout bugs.


## Iteration

go test -v -run TestGenerateReport

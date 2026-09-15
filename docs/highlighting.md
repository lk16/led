# Highlighting

## Keywords

A hardcoded keyword list per language, picked by file extension. No highlighting library like chroma, no grammar files.

Languages: Go (`.go`), JavaScript (`.js`), Python (`.py`), Rust (`.rs`).

A file with any other extension gets no highlighting. led always opens a named file, so the extension is the only thing it goes by. It does not read the first line for a shebang and it does not guess from the content.

A keyword is a whole word: letters, digits and `_` around it make it plain text again. The keyword list holds no context, so a keyword inside a string or a comment is colored too.

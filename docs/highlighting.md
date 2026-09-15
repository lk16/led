# Highlighting

## Keywords

A hardcoded keyword list per language, picked by file extension. No highlighting library like chroma, no grammar files.

## Invalid JSON and YAML

Highlight the first character that makes the file invalid.

YAML is open. The standard library has no YAML parser, and the common YAML library reports the line of an error but not the column. Decide before building it.

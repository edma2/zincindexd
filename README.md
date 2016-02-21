# Overview
zincindexd is a daemon that watches one or more directories that contain
[zinc](https://github.com/typesafehub/zinc) analysis files, maintaining
an in-memory index of fully qualified Scala/Java class names and source
file paths.

zincindexd serves these mappings over the Plan9 [plumber]
(https://en.wikipedia.org/wiki/Plumber_(program)), taking class
names and plumbing the appropriate source files to edit.

# Limitations
zincindexd's file watching implementation uses
[FSEvents](https://en.wikipedia.org/wiki/FSEvents) as the underlying
mechanism. This makes it an OSX-only tool.
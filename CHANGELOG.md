
## 1.1

Changed the call stack data output to be in variables @file_0, @file_1, ... and
@line_0, @line_1, ... to enable more general call stack reporting, with 3
levels included by default.

Replaced the call stack processing with more robust code that uses GOROOT to
exclude Go's own source code files by default. This should mean more useful
output when errors occur in goroutines, such as HTTP handlers.


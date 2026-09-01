package ui

import "embed"

//go:embed html/*.html static/css/style.css static/js/htmx.min.js
var Files embed.FS

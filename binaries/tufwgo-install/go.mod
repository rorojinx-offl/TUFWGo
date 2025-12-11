module tufwgo-install

go 1.25

require github.com/schollz/progressbar/v3 v3.18.0

require (
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
)

require (
	TUFWGo v0.0.0
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/term v0.38.0 // indirect
)

replace TUFWGo => ../..

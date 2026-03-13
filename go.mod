module github.com/freehandle/jornal

go 1.24

toolchain go1.24.3

replace github.com/freehandle/breeze => ../breeze

replace github.com/freehandle/iu => ../iu

replace github.com/freehandle/handles => ../handles

replace github.com/freehandle/safe => ../safe

require (
	github.com/freehandle/breeze v0.0.0-20260203214622-1f5db0dbe8be
	github.com/freehandle/handles v0.0.0-20260204012708-d3a4b57e1412
	github.com/freehandle/iu v0.0.0-20260130030204-5ff6d564599d
)

require (
	github.com/freehandle/papirus v0.0.0-20240109003453-7c1dc112a42b // indirect
	github.com/freehandle/safe v0.0.0-20260224015132-0a952afc0fe3 // indirect
)

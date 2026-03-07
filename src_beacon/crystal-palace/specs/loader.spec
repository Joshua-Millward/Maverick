x64:
	load "../../build/loader.x64.o"
		make pic +gofirst

	run "../../crystal_palace/specs/services.spec"

	run "../../crystal_palace/specs/pico.spec"
		link "pico"

	load "../../build/draugr.x64.o"
		make pic
		export
		preplen
		link "draugr"

	push $CONFIG
		preplen
		link "config"

	generate $KEY 128

	push $DLL
		xor $KEY
		preplen
		link "dll"

	push $KEY
		preplen
		link "mask"

	load "../../build/pic_end.o"
		preplen
		link "pic_end"

	export

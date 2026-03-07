x64:
	load "Bin/obj/main.x64.o"
		make pic +gofirst

		foreach %LIBS: mergelib %_

		load "Bin/obj/entry.x64.o"
			make object
			load "Bin/obj/crypto.x64.o"
				merge
			load "Bin/obj/packer.x64.o"
				merge
			export
			link "entry_module"

		load "Bin/obj/transport.x64.o"
			make object
			mergelib "lib/LibWinHttp/libwinhttp.x64.zip"
			export
			link "transport_module"

		load "Bin/obj/tasks.x64.o"
			make object
			export
			link "task_module"

		load "Bin/obj/obfuscation.x64.o"
			make object
			export
			link "obfuscation_module"

		dfr "resolve" "ror13"
		export

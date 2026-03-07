x64:

  load "../../build/pico.x64.o"
    make object

  load "../../build/callgate.x64.o"
    merge

  load "../../build/hooks.x64.o"
    merge

  load "../../build/kraken.x64.o"
    merge

  load "../../build/guardexec.x64.o"
    merge

  load "../../build/syscalls.x64.o"
    merge

  mergelib "../../crystal_palace/libtcg.x64.zip"

  exportfunc "setup_hooks" "__tag_setup_hooks"
  exportfunc "set_image_info" "__tag_set_image_info"
  exportfunc "set_proxy" "__tag_set_proxy"
  exportfunc "setup_page_guard" "__tag_setup_page_guard"

  addhook "KERNEL32$Sleep" "_Sleep"
  addhook "KERNEL32$ConnectNamedPipe" "_ConnectNamedPipe"
  addhook "KERNEL32$FlushFileBuffers" "_FlushFileBuffers"
  addhook "KERNEL32$WaitForSingleObjectEx" "_WaitForSingleObjectEx"
  addhook "KERNEL32$LoadLibraryA" "_LoadLibrary"
  addhook "KERNEL32$GetProcAddress" "_GetProcAddress"
  addhook "KERNEL32$GetModuleHandleA" "_GetModuleHandleA"

  export

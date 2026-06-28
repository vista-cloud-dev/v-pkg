ZZA3P ;VCD - B.3 e2e pre/post-install fixture ;1.0
 ;;1.0;ZZA3;;
 ; Throwaway fixture for the v-pkg B.3 install-hook live-prove. The pre- and
 ; post-install entryrefs each set a sentinel global so a successful install can
 ; be observed without touching any real VistA file.
 quit
PRE ; pre-install hook (transport "INI") -- PRE^XPDIJ1 D @
 set ^ZZA3OUT("PRE")=1
 quit
POST ; post-install hook (transport "INIT") -- POST^XPDIJ1 D @
 set ^ZZA3OUT("POST")=1
 quit

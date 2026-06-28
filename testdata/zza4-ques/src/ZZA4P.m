ZZA4P ;VCD - A.1.3 e2e QUES-answer fixture ;1.0
 ;;1.0;ZZA4;;
 ; Throwaway fixture for the v-pkg A.1.3 install-question live-prove. The
 ; post-install hook reads a pre-answered build install question via the real
 ; $$ANSWER^XPDIQ and records it in a sentinel global, so the test can confirm the
 ; seeded answer reached the routine (vs the default "" when no answer is seeded).
 quit
POST ; post-install hook (transport "INIT") -- reads the seeded QUES answer
 set ^ZZA4OUT("Q")=$$ANSWER^XPDIQ("ZZA4Q")
 quit

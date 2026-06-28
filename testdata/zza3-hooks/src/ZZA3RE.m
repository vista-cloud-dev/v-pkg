ZZA3RE ;VCD - B.3 e2e environment-check (rejects) ;1.0
 ;;1.0;ZZA3R;;
 ; Rejecting env-check routine (transport "PRE") -- ENV^XPDIL1 D @("^"_name).
 ; Sets the sentinel (proving it ran) AND sets XPDABORT=1, which ENV's ABORT
 ; label returns as 1 -- KIDS refuses the install and purges #9.7. This exercises
 ; A.1.2's env-check ROUTINE arm (the .KID "PRE" node), distinct from the
 ; required-build reject path already proven with the zza2-reqb fixture.
 set ^ZZA3OUT("ENV")=1,XPDABORT=1
 quit

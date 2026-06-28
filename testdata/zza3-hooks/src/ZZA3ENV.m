ZZA3ENV ;VCD - B.3 e2e environment-check (passes) ;1.0
 ;;1.0;ZZA3;;
 ; Env-check routine (transport "PRE") -- ENV^XPDIL1 D @("^"_name). Sets a
 ; sentinel so the live-prove can confirm the env-check actually ran, and does
 ; NOT set XPDABORT, so the install proceeds.
 set ^ZZA3OUT("ENV")=1
 quit

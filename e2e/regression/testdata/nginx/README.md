# NGINX

This is a regression test for <https://github.com/edgelesssys/contrast/issues/624>, ensuring that symlinks with non-ASCII symbols work correctly.
The image used for this test is a Debian bookworm version of NGINX, containing the following symlink:

```console
# ls -l /etc/ssl/certs | grep Aran
lrwxrwxrwx 1 root root     48 Aug 12  2024  988a38cb.0 -> 'NetLock_Arany_=Class_Gold=_F'$'\305\221''tan'$'\303\272''s'$'\303\255''tv'$'\303\241''ny.pem'
lrwxrwxrwx 1 root root     83 Aug 12  2024 'NetLock_Arany_=Class_Gold=_F'$'\305\221''tan'$'\303\272''s'$'\303\255''tv'$'\303\241''ny.pem' -> '/usr/share/ca-certificates/mozilla/NetLock_Arany_=Class_Gold=_F'$'\305\221''tan'$'\303\272''s'$'\303\255''tv'$'\303\241''ny.crt'
```

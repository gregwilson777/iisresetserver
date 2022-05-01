While running automated tests it's useful to reset the IIS server or take database snapshots at certain points...

This simple server provides a way to:
 - remotely trigger an IIS reset after reconfiguring an IIS application server
 - remotely trigger a database snapshot and storage (via a powershell script that must already exist on the system)

To start the server:
```
qamanager -c configfile [-p port]
```
where configfile is a file that just contains a single string which serves as the "password" (only requests with the correct password will be served), 
and port is an optional port number - the default port is 9991.

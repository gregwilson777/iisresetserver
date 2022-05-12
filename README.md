While running automated tests it's useful to reset the IIS server or take database snapshots at certain points...

This simple server provides a way to:
 - remotely trigger an IIS reset after reconfiguring an IIS application server
 - remotely trigger a database snapshot and storage (via a powershell script that must already exist on the system)

To start the server:
```
qamanager -l logfile
```
where logfile is where the logs will be written, duh.

You can add this as a windows service using sc.exe with something like the following commands:

```azure
sc.exe create QAManager type= own start= auto binpath= "C:\QA\qamanager.exe -l C:\QA\qa.log" displayname=QAManager 
sc.exe description QAManager "QAManager service for remote IISReset/DBSnapshots" 
sc.exe failure QAManager actions=restart/60000/restart/60000// reset=86400 
sc.exe start QAManager 
```
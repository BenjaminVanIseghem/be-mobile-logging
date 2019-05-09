# be-mobile-logging

This library is used for limiting the amount of logs that are flushed to a datasource. 
It uses a buffer to store logs temporarily. In the event of an error, fatal, or panic, 
the buffer knows to flush its contents to a file. The name of this file can be anything you want. There are
3 components in defining the name. First its path and then 2 parts of a name. For example: /etc/logs/ + errorlog + 001.

The buffer is stored in a limited array. This way, the buffer get cleared and reused if the program keeps running. 
Same goes for the files. In a Kubernetes environment there can be no cluttering of files. Therefore a max amount of files is
set. If the max amount is reached, the library overwrites and renames the oldest file available.

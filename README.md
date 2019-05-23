# be-mobile-logging

This library is used for limiting the amount of logs that are flushed to a datasource. 
It uses a buffer to store logs temporarily. In the event of an error, fatal, or panic, 
the buffer knows to flush its contents to a fluentd. Fluentd can be configured to catch these logs and forward them to Loki according to their tags.

The buffer is stored in a limited array. This way, the buffer gets cleared and reused if the program keeps running.

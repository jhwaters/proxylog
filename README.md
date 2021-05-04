# proxylog

Creates a server that listens on a TCP port (`-l` address) and connects received clients to a server (`-r` address). All data sent and received over this connection is logged in structured JSON format (`github.com/rs/zerolog`).

## Example

If there is an http server listening on port 8000 of your computer, redirect and log incoming traffic from port 80.

```
> proxylog -l 127.0.0.1:80 -r 127.0.0.1:8000
```

Test with curl.
```
> curl http://127.0.0.1
<!DOCTYPE html>
<html>
  <body>
    <h1>Hello!</h1>
  </body>
</html>
```

`proxylog` output:
```
{"session":"1","clientAddr":"127.0.0.1:49289","serverAddr":"127.0.0.1:8000","time":1619972461106603,"message":"connection established"}
{"session":"1","src":"client","data":"GET / HTTP/1.1\r\nHost: 127.0.0.1\r\nUser-Agent: curl/7.64.1\r\nAccept: */*\r\n\r\n","time":1619972461106754}
{"session":"1","src":"server","data":"HTTP/1.1 200 OK\r\nAccept-Ranges: bytes\r\nContent-Length: 70\r\nContent-Type: text/html; charset=utf-8\r\nLast-Modified: Sun, 02 May 2021 15:21:30 GMT\r\nDate: Sun, 02 May 2021 16:21:01 GMT\r\n\r\n<!DOCTYPE html>\n<html>\n  <body>\n    <h1>Hello!</h1>\n  </body>\n</html>\n","time":1619972461108968}
{"session":"1","clientAddr":"127.0.0.1:49289","serverAddr":"127.0.0.1:8000","time":1619972461109180,"message":"connection closed"}
```

The `session` field increments with every connection but will be reset with each invocation of `proxylog`. A prefix may be added with the `-p` option.

## Command line options

```
> proxylog -h
Usage of proxylog:
  -a	append to log file
  -c	log with console writer
  -l string
    	listen/local address (required)
  -n	do not log data
  -o string
    	log to file instead of stdout
  -p string
    	set session prefix
  -r string
    	remote/server address (required)
  -s	force connections to run synchronously
  -t	log time in iso format
  -v	log listener status
  -x	log bytes in hex format
```

Note that go's `flag` module is used for parsing so rather than `proxylog -tvcx -l :2222 -r 10.0.0.2:22` you will have to do `proxylog -t -v -c -x -l :2222 -r 10.0.0.2:22`.

Required:

- `-l ADDR` listen/local address, e.g. `:80`
- `-r ADDR` remote/server address - address to which traffic should be routed; ideally there should be a server running here to accept the connections

Optional:

- `-o FILE` write logs to this file instead of stdout
- `-p STRING` add a prefix to the connection `session` field

Flags:

- `-a` only applicable if used with `-o`; appends to the log file rather than overwriting it
- `-c` log with `zerolog`'s `ConsoleWriter` rather than JSON
- `-n` disable logging of data sent over connection; logging of connection status will still occur
- `-s` connections received on the `-l` address will not be connected to `-r` until the current connection has closed; by default they will run asynchronously but can be distinguished in the logs by the `session` field
- `-t` log time in ISO format (default is unix microseconds)
- `-v` adds a small amount of additional logging when the listener starts or accepts a connection
- `-x` log data in hex format

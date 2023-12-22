# threaddump-analyzer

Java Thread Dump Analyzer. This is a rewrite of [spotify/threaddump-analyzer](https://github.com/spotify/threaddump-analyzer) in Go. It's mainly for personal use.

Download the lastest one from release page and run it in terminal:

```sh
Usage of threaddump-analyzer-arm64-macos:
  -details
        print all details
  -file string
        path to stack dump
  -report
        output report file including all the details
```

To create a detailed report:

```sh
./threaddump-analyzer-arm64-macos -file mydump.txt -report

# Created report 'mydump_report.txt' for dump 'mydump.txt'
```

To view the summary in terminal:

```sh
./threaddump-analyzer-arm64-macos -file mydump.txt

# Summary:
#
# In total 1644 threads found
#
#        Curator-TreeCache                                                            : 271 threads with similar names (16.484%)
#        ReconcileService                                                             : 271 threads with similar names (16.484%)
#
#        ...
#
#        redisson-3                                                                   : 16  threads with similar names (0.973%)
#        http-nio-9405-exec                                                           : 10  threads with similar names (0.608%)
#        PollingServerListUpdater                                                     : 2   threads with similar names (0.122%)
#        com.alibaba.nacos.client.Worker                                              : 2   threads with similar names (0.122%)
#
#        Remaining                                                                    : 695 threads (42.275%)
```
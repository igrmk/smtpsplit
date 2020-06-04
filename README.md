smtpsplit
=========

[![Build Status](https://travis-ci.org/igrmk/smtpsplit.png)](https://travis-ci.org/igrmk/smtpsplit)
[![GoReportCard](http://goreportcard.com/badge/igrmk/smtpsplit)](http://goreportcard.com/report/igrmk/smtpsplit)

This is simple SMTP router and splitter. It routes incoming traffic depending on a recepient domain.
Currently it supports STARTTLS only for incoming connections.

Usage
-----

1. Create a configuration file. Here is an example
   ```
   {
       "listen_address": ":25",
       "routes": {
           "xxx.com": "localhost:2500",
           "yyy.com": "localhost:2600"
       }
   }
   ```

2. Run `smtpsplit your_config.json`

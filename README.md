smtpsplit
=========

[![Build Status](https://travis-ci.org/igrmk/smtpsplit.png)](https://travis-ci.org/igrmk/smtpsplit)
[![GoReportCard](http://goreportcard.com/badge/igrmk/smtpsplit)](http://goreportcard.com/report/igrmk/smtpsplit)

This is simple SMTP router and splitter. It routes the incoming traffic depending on a recepient domain.
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

Configuration
-------------

<dl>

<dt>listen_address</dt>
<dd>the address to listen to for incoming emails</dd>

<dt>host</dt>
<dd>the host name used to introduce this router</dd>

<dt>timeout_seconds</dt>
<dd>the timeout for incoming and outgoing emails</dd>

<dt>debug</dt>
<dd>debug mode</dd>

<dt>certificate</dt>
<dd>the certificate path for STARTTLS</dd>

<dt>certificate_key</dt>
<dd>the certificate key path for STARTTLS</dd>

<dt>routes</dt>
<dd>a domain to an address map</dd>

</dl>

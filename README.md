Setup
=====
1. Log into http://panel.webfaction.com
2. Add a domain, for example: mydomain.example.com (no need to set up further)

Usage
======
1. Rename updateip.json.example to updateip.json
2. Modify update.json to add your Webfaction username, password, and the domain
   in your setup
3. Compile program with "go build updateip.go" and put the binary and config
   file in the same directory
4. Add to cron:
   ` */7 * * * * cd /Users/example/bin/ && /Users/example/bin/updateip > /dev/null 2>&1`

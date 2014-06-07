Export vBulletin to JSON
========================

This utility will export a vBulletin instance to static JSON files.

The JSON files can then be used to archive a vBulletin site, or as the source of an import into another forum system or community CMS.

Preparing to export
-------------------

The export is performed with some basic assumptions in place:

1. We will export from a standalone database
2. We have no access to your PHP files
3. We have no access to web server file system
4. We have no access to your production servers

With that in mind, we expect that you have:

1. Configured vBulletin to store attachments, avatars, etc in the database rather than the file system and moved them to the database if necessary.
2. Performed a MySQL dump
3. Restored the dumped database to a local/dev instance of MySQL and you are *NOT* running this on a production server

When the export task runs it will go as fast as your system allows and you should not be running this on a production system.

**If you run this on your production system, it will bring your site down.**

Once these steps have been completed and you have a local MySQL instance running just the vBulletin database (and that includes attachments and custom avatars), then you are ready to begin your export.

Running the export
------------------

1. Ensure you have [Go installed](http://golang.org/doc/install) and `$GOPATH` defined
2. `go get -u github.com/microcosm-cc/export-vbulletin`
3. `cd $GOPATH/src/github.com/microcosm-cc/export-vbulletin`
4. `make`
5. Edit `config.toml` to give the connection details to the local MySQL database
5. `export-vbulletin`

Then wait. it can take a *long* time depending on the size of your forum and the time depends on the capabilities of the local system, especially the disk speed and amount of RAM.

As a guide, a forum with 40k users, 4.5m posts, 1.5m private messages and 70k attachments took under an hour to export on a server spec workstation (with fast disks), and just over 3 hours to export on a few year old laptop (with SSD).

During the export, progress of each task will be printed to the console window.

Limitations
-----------

Not all data is exported as certain pieces of information are only relevant to the internal workings of vBulletin.

*Reputation*: Not exported as any system that imports would have to perfectly reproduce how vBulletin calculated it for it to have any future value. It is expected that a destination system would have it's own reputation system and be able to calculate relevant values during or after an import.

*Forum Hierarchies*: Not exported as different software treats containers differently, some consider forums as a flat list, and some consider forums as labels or categories on content.

*Private Message BCC*: Exported as a standard To recipient as not all systems have the concept of a BCC.

Troubleshooting
---------------

Things that can go wrong:

1. *Not enough disk space to perform the export*. Please ensure you have more than double what you think will need before starting. i.e. for an exported database that takes 20GB you should have more than 40GB in free disk space when you start the export. However that is a minimum requirement and we'd recommend you don't cut it so close.
2. *Export fails with an error message*. In every case we came across this is due to not having enough resources to run as many processes as we'd like against the database. This either results in exhausting the database connections, exhausting file system descriptors, or some other similar issue. In the `config.toml` file you can reduce the `Connections` parameter. A value of `1` will process all data sequentially, a value of `2` will create two child processes and run in parallel. If you have trouble a value below 4 is recommended, if you're on a well-spec'd server a value of 50 is possible.

Exports are resumable. If you encounter any issues, simply resolve the issue and re-run the export job. The export job will skip all items previously exported successfully.


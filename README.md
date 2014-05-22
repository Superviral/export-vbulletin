Export vBulletin
================

Exports a vBulletin created MySQL database (with attachments loaded into the
database) into a set of directories and JSON files.

Preparing to export
-------------------

Some basic assumptions exist, that we are about to perform an export from a
standalone database and have no access to your PHP files, file storage, or
production servers.

We are going to hammer the database, and so we assume that you have:

1. Configured vBulletin to store attachments, avatars, etc in the database
   rather than the file system and moved them to the database if necessary.
2. Performed a MySQL dump
3. Restored the dumped database to a local/dev instance of MySQL and you are
   *NOT* running this on a production server

Once these steps have been completed the export-vbulletin binary should be
placed in a directory along with config.toml 

config.toml should be edited to reflect the connection details to the local/dev
database, and to declare the params for the exported data.

Running the export
------------------

./export-vbulletin

Then wait... it can take a *long* time depending on the size of your forum.
Updates will be printed to the console window.

Assumptions
-----------

Not all data is exported as certain pieces of information are only relevant to
the internal workings of vBulletin.

*Reputation*: Not exported as any system that imports would have to perfectly
reproduce how vBulletin calculated it for it to have any future value.

*Forum Hierarchies*: Not exported as different software treats containers
differently, some consider forums as a flat list, and some consider forums as
labels or categories on content.

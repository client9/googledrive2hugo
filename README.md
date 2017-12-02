# googledrive2hugo
Converts google docs to Hugo markdown  WIP

The Goal is to edit Google Docs content in Google Drive and then use Hugo (and friends) to publish it

We do this by:

* Reading a folder of Google Docs off a Google Drive
* Convert to markdown with Hugo front matter
* Place output in a Hugo site directory

From there you publish in a few ways

* Run Hugo and serve directly or publish output
* Commit the generated markdown pages, trigger Travis-CI, whatever


It's not clear what exactly the best API is since I'm new to Google Drive API

# MABLE (Making the ABL Easier)

MABLE is a command line application built with Go to edit the OpenStax Approved Book List (ABL)

### Getting Mable

If you are using a mac, the precompiled binary in the repo will work, you don't need to build anything.

Currently, only the standard Go libraries are used. To build MABLE, download the repo, make sure you have Go installed, and run `go build` in the project directory. This will create the `mable` binary file. 

### Using Mable

* For help, run `./mable -h` in the terminal.
* To retrieve the most recent version of the ABL, run `./mable -update`
* To remove a book version from the ABL, run `./mable -remove {collection id} {content version}`
* To add a book version to the ABL, run `./mable -add {collection id} {content version} {min code version}`
* These changes are currently made to the local `approved-book-list.json` file. To add the changes to the OpenStax ABL, you must use regular git commands (or just cut and paste it over the old version).

### To be added

* Add new book to the ABL (instead of only book versions)
* Push changes to github from MABLE


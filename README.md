
# mviplayer

This is a simple program to tidy up radio shows downloaded from BBC iPlayer.

Example:

    mviplayer *.m4a ~/Radio/

The tags from the m4a files are read and used to rename according to the pattern:

    Show Name/sNN eMM Title of Episode.m4a

So in the example, I might end up with `~/Radio/The Goon Show/s07 e15 Wings Over Dagenham.m4a`

To test how things will be renamed before renaming them, use the `-t` option.

To report verbosely what's done, use the `-v` option.

You can add custom rename rules if the default behavior doesn't work for particular shows. They are held in a file `rules.json` in the destination directory. Example:

```json
[
  {"from": ":?\\s+Series\\s+\\d+", "to": ""},
  {"from": "Quanderhorn: Quanderhorn", "to": "Quanderhorn"}
]
```

The first rule renames anything called `Show Name Series 2` (and so on) to just the show name.

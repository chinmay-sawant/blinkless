# Image command

`make build` writes `bin/blinkless` from `cmd/blinkless`.

`main` calls `cli.Parse(argv, cli.ModeImage)` and then `app.RunImage`. The command does not import a PDF writer.

Useful flags:

- `-o` output path
- `--html`, `--url`, or a positional HTML file
- `--allow-local-files` and `--allow` for local and remote resources
- `--width`, `--height`, `--format png|jpg`, `--quality`
- `--font-path`, `--use-system-fonts`
- `--zoom`, `--background`, `--media-type`

`--version` prints `internal/cli.Version`. The Makefile stamps that variable as `0.0.1`. There is no `VERSION` file.

`make samples` runs this binary once per golden body fixture and writes `output/<fixture>.png`.

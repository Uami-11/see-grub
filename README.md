# see-grub

Made a grub theme viewer because grub2-theme-preview just does not work for me and I would have to reboot everytime to see my theme. It parses your theme's `theme.txt`, validates theme files, and renders a live preview window.

Made with go and the power of [Ebitengine](https://ebitengine.org/)!

## Installation

### From source

```sh
git clone https://github.com/Uami-11/see-grub
cd see-grub
go build -o see-grub .
```

Requires Go 1.26 or later.

## Usage

```sh
see-grub <theme-directory-or-file> [--gfxmode=WxH] [options]
```

### Options

| Flag | Description |
|---|---|
| `--gfxmode=WxH` | Set preview window to a specific width and height in pixels. |
| `--currentEntries` | List the current menu entries and exit. |
| `--changeEntries` | Interactively modify an existing menu entry. |
| `--addEntry[=TEXT]` | Add a new menu entry. If TEXT is provided inline it is used directly; otherwise you will be prompted. |
| `--resetEntries` | Reset menu entries back to the default three. |
| `--help`, `-h` | Show the help message. |

### Examples

```sh
# Preview a theme directory
see-grub ~/.grub/themes/my-theme

# Preview a specific theme.txt file
see-grub ~/Downloads/theme/theme.txt

# Preview at a custom resolution (e.g. your screen's native resolution)
see-grub ~/.grub/themes/my-theme --gfxmode=2560x1600

# Add a custom boot entry inline
see-grub --addEntry="My Custom Entry"

# Add a boot entry interactively
see-grub --addEntry

# Modify an existing entry
see-grub --changeEntries

# Reset entries to defaults
see-grub --resetEntries
```

### Controls

| Key | Action |
|---|---|
| `↑` / `↓` | Navigate boot entries |
| `ESC` / `Q` | Quit preview |

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
see-grub <theme-directory-or-file> [--gfxmode=WxH]
```

### Examples

```sh
# Preview a theme directory
see-grub ~/.grub/themes/my-theme

# Preview a specific theme.txt file
see-grub ~/Downloads/theme/theme.txt

# Preview at a custom resolution (e.g. your screen's native resolution)
see-grub ~/.grub/themes/my-theme --gfxmode=2560x1600
```

### Controls

| Key | Action |
|---|---|
| `↑` / `↓` | Navigate boot entries |
| `ESC` / `Q` | Quit preview |

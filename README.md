<p align="center">
  <img src="assets/icon.png" width="128" alt="huectl">
  <h1 align="center">huectl</h1>
  <p align="center">Control Philips Hue lights from your terminal</p>
</p>

<p align="center">
  <a href="https://github.com/Aayush9029/huectl/releases/latest"><img src="https://img.shields.io/github/v/release/Aayush9029/huectl" alt="Release"></a>
  <a href="https://github.com/Aayush9029/huectl/blob/main/LICENSE"><img src="https://img.shields.io/github/license/Aayush9029/huectl" alt="License"></a>
</p>

## Install

```bash
brew install aayush9029/tap/huectl
```

Or tap first:

```bash
brew tap aayush9029/tap
brew install huectl
```

## Usage

```bash
huectl             # interactive dashboard
huectl auth
huectl status
huectl on
huectl on 2 -b 180
huectl off all
huectl toggle "lamp 1"
huectl color desk ff8800
huectl color all blue --no-on
```

`huectl auth` stores the Hue Bridge app key locally in `~/.config/huectl/config.json`.

Colors can be hex (`ff8800` or quoted `"#ff8800"`), basic names like `blue`,
or `rgb:r,g,b` / `hsv:h,s,v`.

## License

MIT

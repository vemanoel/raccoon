sudo zypper install --no-confirm --no-recommends telegram-desktop
sudo zypper install --no-confirm --no-recommends deno yt-dlp
sudo zypper install --no-confirm --no-recommends git go gopls
sudo zypper install --no-confirm --no-recommends iosevka-fonts

sudo zypper addrepo https://download.opensuse.org/repositories/editors/openSUSE_Tumbleweed/editors.repo
sudo zypper --gpg-auto-import-keys refresh editors
sudo zypper install --no-confirm --no-recommends zed

sudo zypper addrepo https://brave-browser-rpm-release.s3.brave.com/brave-browser.repo
sudo zypper --gpg-auto-import-keys refresh brave-browser
sudo zypper install --no-confirm --no-recommends brave-origin

sudo zypper install --no-confirm --no-recommends kora-icon-theme
xfconf-query -c xsettings -p /Net/IconThemeName -s "kora"

xfconf-query -c xsettings -p /Net/ThemeName -s "Adwaita-dark"

xfconf-query -c xfwm4 -p /general/theme -s "Greybird-geeko-dark"

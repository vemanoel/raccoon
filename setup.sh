sudo zypper install --no-confirm --no-recommends --force telegram-desktop
sudo zypper install --no-confirm --no-recommends --force deno yt-dlp
sudo zypper install --no-confirm --no-recommends --force git go gopls
sudo zypper install --no-confirm --no-recommends --force iosevka-fonts

sudo zypper install --no-confirm --no-recommends MozillaFirefox libopenh264-8

sudo zypper addrepo https://download.opensuse.org/repositories/editors/openSUSE_Tumbleweed/editors.repo
sudo zypper --gpg-auto-import-keys refresh editors
sudo zypper install --no-confirm --no-recommends --force zed

sudo zypper addrepo https://brave-browser-rpm-release.s3.brave.com/brave-browser.repo
sudo zypper --gpg-auto-import-keys refresh brave-browser
sudo zypper install --no-confirm --no-recommends --force brave-origin

sudo zypper install --no-confirm --no-recommends --force kora-icon-theme
xfconf-query -c xsettings -p /Net/IconThemeName -s "kora"

xfconf-query -c xsettings -p /Net/ThemeName -s "Adwaita-dark"

xfconf-query -c xfwm4 -p /general/theme -s "Greybird-geeko-dark"

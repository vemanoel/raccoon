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

sudo zypper install git gtk2-engine-murrine sassc

sudo zypper install --no-confirm --no-recommends  kora-icon-theme

mkdir -p ${HOME}/.local/share/themes
sudo zypper install --no-confirm --no-recommends  gtk2-engine-murrine sassc
git clone https://github.com/vinceliuice/Fluent-gtk-theme.git
cd Fluent-gtk-theme
./install.sh --size compact --color dark --theme grey --icon opensuse --tweaks noborder --dest ${HOME}/.local/share/themes

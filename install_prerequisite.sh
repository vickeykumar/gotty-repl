#!/usr/bin/env bash
# please run as sudo
# reboot once after installing to take effect for cgroups and containers

retVal=0

cd ~

export DEBIAN_FRONTEND=noninteractive
export TZ=Etc/UTC

apt-get update -y
apt-get install -y --no-install-recommends make gcc g++ default-jdk git

# set cap to nsenter,gcc,g++
apt-get install -y --no-install-recommends libcap2-bin
setcap "cap_sys_admin,cap_sys_ptrace+ep" /usr/bin/nsenter 
chown root.root /usr/bin/nsenter
chmod 4755 /usr/bin/nsenter
# && ./containers-from-scratch/main run 2 nsenter -n -t$$ /bin/bash
# sudo setcap "cap_sys_admin,cap_sys_ptrace+ep" /usr/bin/arm-linux-gnueabihf-gcc-8
# add export NODE_OPTIONS=--max_old_space_size=2048 to .bashrc
apt-get install -y --no-install-recommends net-tools

#install golang
apt-get install -y --no-install-recommends golang

#install yaegi go repl, >= go1.18
go install github.com/traefik/yaegi/cmd/yaegi@latest
apt-get install -y --no-install-recommends yaegi

#install npm
apt-get install -y --no-install-recommends npm
#install the last stable release of npm and node for this project using nvm
# node v12.22.9
cd ~
apt-get -y --no-install-recommends install curl wget bzip2
echo "home: " $HOME
curl -sL https://raw.githubusercontent.com/nvm-sh/nvm/v0.35.0/install.sh -o $HOME/install_nvm.sh 
chmod 755 $HOME/install_nvm.sh
source $HOME/install_nvm.sh
source $HOME/.bashrc

# ENV for nvm
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm
nvm install v12.22.9
npm install npm@8.5.1 -g

#install cling
#use following to compile minimal cling
#/usr/bin/ld.gold --strip-all --no-map-whole-files --no-keep-memory --no-keep-files-mapped $@ 
# cling takes some time to init first instance, add below lines to rc.local(startup)
#/usr/bin/cling 21321 .q > /dev/null 2>&1 &
cd ~
wget https://raw.githubusercontent.com/vickeykumar/openrepl/e596c6f0918e48eeba7a0bf7b7d2632f6b155ffb/repls/cling-Ubuntu-22.04-x86_64-1.0~dev-d47b49c.tar.bz2
tar -xvf cling-Ubuntu-22.04-x86_64-1.0~dev-d47b49c.tar.bz2
chmod 755 cling-Ubuntu-22.04-x86_64-1.0~dev-d47b49c/bin/cling
ln -s $HOME/cling-Ubuntu-22.04-x86_64-1.0~dev-d47b49c/bin/cling /usr/local/bin/cling
rm cling-Ubuntu-22.04-x86_64-1.0~dev-d47b49c.tar.bz2

#install gointerpreter
git clone https://github.com/vickeykumar/Go-interpreter.git
cd Go-interpreter
make install
cd ..

#install ipython2.7
apt-get install -y --no-install-recommends python2.7
ln -s /usr/bin/python2.7 /usr/bin/python
apt-get install -y --no-install-recommends ipython
apt-get install -y --no-install-recommends python-is-python3

#install ipython3
apt-get install -y --no-install-recommends ipython3

#install Ruby(irb)
apt-get install -y --no-install-recommends ruby

#install perli
apt-get install -y --no-install-recommends rlwrap
# append alias to the /etc/profile
# alias yaegi="rlwrap yaegi"

apt-get install -y --no-install-recommends perl
apt-get install -y --no-install-recommends perl-doc
git clone https://github.com/vickeykumar/perli.git
cd perli && make install
cd ~

# Docker: Error response from daemon: cgroups: cgroup mountpoint does not exist: unknown
# if using docker use privileged mode with cgroup mounted (-v /sys/fs/cgroup:/sys/fs/cgroup:rw )
mkdir /sys/fs/cgroup/systemd
mount -t cgroup -o none,name=systemd cgroup /sys/fs/cgroup/systemd

apt-get install -y --no-install-recommends gdb


#cleanup
apt-get -y clean
apt-get -y autoremove

#test
test_commands=(
	"gcc --version"
	"g++ --version"
	"cling --version"
	"go version"
	"yaegi version"
	"python2.7 --version"
	"python3 --version"
	"python --version"
	"ipython3 --version"
	"irb --version"
	"ruby --version"
	"perl --version"
	"perli --version"
	"java --version"
)


# Loop through the array and execute each command
for cmd in "${test_commands[@]}"; do
  $cmd
  if [ $? -eq 0 ]; then
    echo "test [$cmd] => PASSED"
  else
    echo "test [$cmd] => FAILED with status code $?"
    retVal=1
  fi
done

exit $retVal

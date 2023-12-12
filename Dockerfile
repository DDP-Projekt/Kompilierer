FROM ubuntu:22.04
#golang is not recent enough in the official ubuntu-repos
RUN apt-get update &&  \
    apt-get install -y git make llvm-12-dev curl
RUN curl https://raw.githubusercontent.com/canha/golang-tools-install-script/master/goinstall.sh | bash
ADD . /Kompilierer
CMD ["bash"]

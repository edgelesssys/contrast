FROM ubuntu:22.04@sha256:340d9b015b194dc6e2a13938944e0d016e57b9679963fdeb9ce021daac430221

# Install requirements
RUN apt-get update && apt-get install -y software-properties-common &&\
    apt-add-repository ppa:zanchey/asciinema && apt-get update &&\
    apt-get install -y curl jq expect asciinema sudo unzip &&\
    rm -rf /var/lib/apt/lists/*

RUN curl -fsSLO https://dl.k8s.io/release/v1.26.0/bin/linux/amd64/kubectl &&\
    sudo install kubectl /usr/local/bin/kubectl && rm kubectl

# As mount point for $HOME/.kube/config
RUN mkdir /root/.kube

# Enable RGB colors in PS1
ENV TERM=xterm-256color
# Set width of terminal, default is ~80 and leads to broken lines for long lines,
# e.g., curl & cosign commands.
ENV COLUMNS=512
# For PS1 to work shell needs to specified
ENV SHELL=/bin/bash
# ANSI color codes are used to control PS1 prompt. We use "\033[38;2;<r>;<g>;<b>m"
# to control the foreground color with RBG colors [1]. Non-printable characters
# need to  be escaped with additional \[ and \], see [2].
# [1]: https://stackoverflow.com/a/33206814/2306355
# [2]: https://stackoverflow.com/a/19501528/2306355
RUN echo 'export PS1="\[\033[38;2;139;4;221m\]$\[\033[0m\] "' >> /root/.bashrc

WORKDIR /demo
ENTRYPOINT ["/usr/bin/expect", "-f"]

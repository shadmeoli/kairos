FROM golang:1.25-bookworm

ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=${USER_UID}

ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Etc/UTC

RUN apt-get update     && apt-get install -y --no-install-recommends         ca-certificates         curl         git         make         openssh-client         procps         sudo         zsh     && rm -rf /var/lib/apt/lists/*

RUN if ! getent group "${USER_GID}" >/dev/null; then groupadd --gid "${USER_GID}" "${USERNAME}"; fi     && if ! id -u "${USERNAME}" >/dev/null 2>&1; then useradd --uid "${USER_UID}" --gid "${USER_GID}" -m -s /usr/bin/zsh "${USERNAME}"; fi     && echo "${USERNAME} ALL=(root) NOPASSWD:ALL" > "/etc/sudoers.d/${USERNAME}"     && chmod 0440 "/etc/sudoers.d/${USERNAME}"

USER ${USERNAME}
ENV HOME=/home/${USERNAME}
ENV ZSH=/home/${USERNAME}/.oh-my-zsh

RUN sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended     && sed -i 's/^ZSH_THEME=.*/ZSH_THEME="robbyrussell"/' "${HOME}/.zshrc"     && sed -i 's/^plugins=.*/plugins=(git golang)/' "${HOME}/.zshrc"

WORKDIR /workspaces/kairos

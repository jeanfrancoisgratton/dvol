%define debug_package   %{nil}
%define _build_id_links none
%define _name dvol
%define _prefix /opt
%define _bash_completionsdir /usr/share/bash-completion/completions
%define _zsh_completionsdir  /usr/share/zsh/site-functions
%define _version 2.25.00
%define _rel 0
%define _binaryname dvol

Name:       dvol
Version:    %{_version}
Release:    %{_rel}
Summary:    Docker/Podman volume backup & restore tool

Group:      CI/CD
License:    GPL2.0
URL:        https://git.famillegratton.net:3000/devops/dvol.git

Source0:    %{name}-%{_version}.tar.gz
#BuildArchitectures: x86_64
BuildRequires: gcc
Recommends: zsh
Requires: bash-completion

%description
Nexus Repository Manager tools

%prep
%autosetup

%build
cd src
go mod download
PATH=$PATH:/opt/go/bin CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -buildid=" -o %{_builddir}/%{_binaryname} .

%clean
rm -rf $RPM_BUILD_ROOT

%pre
%install
install -Dpm 0755 %{_builddir}/%{_binaryname} %{buildroot}%{_bindir}/%{_binaryname}

%post
# Bash completion — always install
/opt/bin/dvol completion bash > %{_bash_completionsdir}/dvol

# Zsh completion — only if zsh is present
if command -v zsh > /dev/null 2>&1; then
    mkdir -p /usr/share/zsh/site-functions
    /opt/bin/dvol completion zsh > /usr/share/zsh/site-functions/_dvol
    zsh -c 'autoload -Uz compinit && compinit' 2>/dev/null || true
fi

%preun

%postun
if [ $1 -eq 0 ]; then
    # $1 == 0 means this is a full uninstall, not an upgrade
    rm -f %{_bash_completionsdir}/dvol
    rm -f %{_zsh_completionsdir}/_dvol
fi

%files
%defattr(0755,root,root,-)
%{_bindir}/%{_binaryname}


%changelog
* Thu Jun 04 2026 Binary package builder <builder@famillegratton.net> 2.25.00-0
- fixed mistyped filename
- Removed missing install scripts
- Upgraded the helperFunctions package from v3 to v5
- fixed rootdir volume issue when restoring; added archlinux packaging support, refactored redhat packaging support
- Merge pull request 'Merge pull request 'develop' (#1) from develop into main' (#2) from main into develop
- Merge pull request 'develop' (#1) from develop into main
- fixed alpine build scripts
- Interim commit while we uniformize all log groups

* Fri Oct 24 2025 Binary package builder <builder@famillegratton.net> 2.10.10-1
- Release number bump to avoid tito messiness (jean-
  francois@famillegratton.net)

* Fri Oct 24 2025 Binary package builder <builder@famillegratton.net> 2.10.10-0
- Completed restore verbosity (jean-francois@famillegratton.net)
- Automatic commit of package [dvol] release [2.10.10-0].
  (builder@famillegratton.net)
- Updated GO, completed verbosity (jean-francois@famillegratton.net)
- Completed verbosity enhancements in backup subcommand (jean-
  francois@famillegratton.net)
- Completed verbosity on the backup command (jean-francois@famillegratton.net)

* Fri Oct 24 2025 Binary package builder <builder@famillegratton.net>
- Completed restore verbosity (jean-francois@famillegratton.net)
- Automatic commit of package [dvol] release [2.10.10-0].
  (builder@famillegratton.net)
- Updated GO, completed verbosity (jean-francois@famillegratton.net)
- Completed verbosity enhancements in backup subcommand (jean-
  francois@famillegratton.net)
- Completed verbosity on the backup command (jean-francois@famillegratton.net)

* Fri Oct 24 2025 Binary package builder <builder@famillegratton.net>
- Completed restore verbosity (jean-francois@famillegratton.net)

* Thu Oct 23 2025 Binary package builder <builder@famillegratton.net> 2.10.10-0
- Updated GO, completed verbosity (jean-francois@famillegratton.net)
- Completed verbosity enhancements in backup subcommand (jean-
  francois@famillegratton.net)
- Completed verbosity on the backup command (jean-francois@famillegratton.net)

* Wed Oct 08 2025 Binary package builder <builder@famillegratton.net> 2.10.00-0
- package version bump (jean-francois@famillegratton.net)
- Interim attempt at building a valid package (jean-
  francois@famillegratton.net)
- Added verbosity (jean-francois@famillegratton.net)
- first commit (jean-francois@famillegratton.net)

* Thu Jun 19 2025 APK Builder <builder@famillegratton.net> 1.10.00-0
- Fixed backup and restore (jean-francois@famillegratton.net)
- refactored subpackage (jean-francois@famillegratton.net)
- Removed un-needed function (jean-francois@famillegratton.net)
- Backup is fixed, working on restore (jean-francois@famillegratton.net)
- Removed some leftover hardcoded API version (jean-
  francois@famillegratton.net)

* Wed Jun 11 2025 APK Builder <builder@famillegratton.net> 1.05.00-0
- Doc update (jean-francois@famillegratton.net)
- Tool version bump (jean-francois@famillegratton.net)
- code is now api version agnostic (jean-francois@famillegratton.net)
- Globalized most variables (jean-francois@famillegratton.net)
- changed -H behaviour (jean-francois@famillegratton.net)

* Tue Jun 10 2025 APK Builder <builder@famillegratton.net> 1.02.00-0
- Added volume listing function (jean-francois@famillegratton.net)
- updated RPM deps script (builder@famillegratton.net)

* Mon Jun 09 2025 APK Builder <builder@famillegratton.net> 1.01.00-1
- new package built with tito

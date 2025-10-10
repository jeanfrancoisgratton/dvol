%ifarch aarch64
%global _arch aarch64
%global BuildArchitectures aarch64
%endif

%ifarch x86_64
%global _arch x86_64
%global BuildArchitectures x86_64
%endif

%define debug_package   %{nil}
%define _build_id_links none
%define _name dvol
%define _prefix /opt
%define _version 2.10.01
%define _rel 0
#%define _arch x86_64
%define _binaryname dvol

Name:       dvol
Version:    %{_version}
Release:    %{_rel}
Summary:    Container volume backup and restore

Group:      Container utilities
License:    GPL2.0
URL:        https://git.famillegratton.net:3000/devops/dvol.git

Source0:    %{name}-%{_version}.tar.gz
BuildRequires: gcc

%description
Container volume backup and restore

%prep
%autosetup

%build
cd %{_sourcedir}/%{_name}-%{_version}/src
PATH=$PATH:/opt/go/bin go build -o %{_sourcedir}/%{_binaryname} .
strip %{_sourcedir}/%{_binaryname}

%clean
rm -rf $RPM_BUILD_ROOT

%pre
exit 0

%install
install -Dpm 0755 %{_sourcedir}/%{_binaryname} %{buildroot}%{_bindir}/%{_binaryname}

%post

%preun

%postun

%files
%defattr(-,root,root,-)
%{_bindir}/%{_binaryname}


%changelog
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


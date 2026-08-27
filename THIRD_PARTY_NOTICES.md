# Third-Party Notices

## VcXsrv

Windows packages of Bashes include VcXsrv 21.1.16.1 as a separate process for
optional SSH X11 forwarding.

- Project: https://github.com/marchaesen/vcxsrv
- Source: https://github.com/marchaesen/vcxsrv/tree/21.1.16.1
- License: GNU General Public License version 3

The Windows package includes the upstream `COPYING` file. Bashes does not load
or link VcXsrv as a library; it starts the bundled executable only when X11
forwarding is explicitly enabled for an SSH session.

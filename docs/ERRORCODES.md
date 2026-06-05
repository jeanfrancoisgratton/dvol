| Category               | Code  | Meaning                                 |
|------------------------|-------|-----------------------------------------|
| JSON                   | 101   | Error rendering JSON message            |
| JSON                   | 102   | Error decoding JSON payload             |
| JSON                   | 103   | Error encoding JSON payload             |
| ImagePull              | 201   | Error pulling image                     |
| ContainerListVolumes   | 301   | Error list containers using volumes     |
| ContainerStart         | 302   | Error starting container                |
| ContainerStop          | 303   | Error stopping container                |
| ContainerCreate        | 304   | Error creating temp container           |
| ContainerArchiveGet    | 401   | Error fetching container archive        |
| ContainerArchiveCreate | 402   | Error creating archive                  |
| ContainerArchiveRead   | 403   | Error reading archive                   |
| ContainerArchiveHeader | 404   | Error writing archive header            |
| ContainerArchiveData   | 405   | Error writing archive data              |
| APIVersion             | 501   | API version negotiation error           |
| APIVersion             | 502   | API version negotiation non-fatal error |
| APIVersion             | 503   | API version field missing from response |
| VolumeDelete           | 601   | Error creating volume delete request    |
| VolumeDelete           | 602   | Error executing volume delete request   |
| VolumeCreate           | 603   | Error re-creating the volume            |
| VolumeCreate           | 604   | Error re-creating the volume            |
| TarballRead            | 701   | Error reading tarball                   |
| TarballRead            | 702   | Error decompressing tarball             |             
| Restore                | 801   | Error creating restore http request     |
| Restore                | 802   | Error executing restore http request    |
| Restore                | 803   | Error reading restore http request      |

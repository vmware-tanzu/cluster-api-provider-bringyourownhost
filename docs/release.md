# Release
This document describes how to release the Cluster API BYOH infrastructure provider.

---
### Follow the below steps to make a new release in BYOH
1. Open a PR to update `cluster-api-provider-bringyourownhost/metadata.yaml` file if a new major/minor version is being released. This is not needed in case of patch release. Map the newly added version to supported Cluster API contract version.

2. Create a new git tag from the main branch of the upstream git repo(vmware-tanzu/cluster-api-provider-bringyourownhost). For example - `git tag v0.3.0`. We follow semantic versioning for releases, see https://semver.org.

3. Push the git tag to upstream repo(vmware-tanzu/cluster-api-provider-bringyourownhost) repository
`git push v0.3.0`

4. This will trigger a new Github Actions workflow called [draft-release](https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/actions/workflows/draft-release.yaml). Wait for the workflow to finish. This workflow will create a draft release having the contributions, changes and assets. Verify the assets, they should contain BYOH agent, cluster templates and infrastructure components.

5. Next step is to release `BYOH` controller OCI image.  Create controller image in local using below make target
```shell
IMG=projects.registry.vmware.com/cluster_api_provider_bringyourownhost/cluster-api-byoh-controller:v0.3.0 make docker-build
```
   This will create an image having name - `projects.registry.vmware.com/cluster_api_provider_bringyourownhost/cluster-api-byoh-controller`
   
   TODO: This is currently a manual step. Needs to be automated.

6. Next, this image needs to be pushed to the `projects.registry.vmware.com` harbor registry. The project maintainers have access to the harbor registry, so they can be contacted to perform this step.
   
```shell
docker login projects.registry.vmware.com -u <username> -p <password>
docker push projects.registry.vmware.com/cluster_api_provider_bringyourownhost/cluster-api-byoh-controller:v0.3.
```

7. Verify and test the release artifacts. 

8. Publish the draft release using the `Publish Release` option in Github UI. 

9. Update the documentation, if applicable.

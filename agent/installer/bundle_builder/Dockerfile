# Usage
#
# Build and push a BYOH bundle to repository
#
# 1. Download bundle ingredients
#    	containerd, kubeadm, kubelet, kubectl
# 2. Mount the bundle ingredients under /ingredients
# 3. Optional. Mount additional configuration under /config
#	-v config/ubuntu/20_04/k8s/1_22:/config
#	Defaults to config/ubuntu/20_04/k8s/1_22
# 4. Build and push a BYOH bundle to repository
#       docker run --rm -v <INGREDIENTS_HOST_ABS_PATH>:/ingredients <THIS_IMAGE> <REPO>/<BUNDLE IMAGE>

FROM harbor-repo.vmware.com/dockerhub-proxy-cache/k14s/image

WORKDIR /bundle-builder
COPY build-payload.sh push-bundle.sh build-push-bundle.sh ./
RUN chmod a+x *.sh
#Default config
COPY config/ubuntu/20_04/k8s/ /config/

RUN mkdir /ingredients
ENV PATH="/bundle-builder:${PATH}"

WORKDIR /bundle
ENTRYPOINT ["build-push-bundle.sh", "/ingredients", "/config"]

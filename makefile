build:
	go build .
	docker build -t fsc --no-cache .
dep:
	kubectl apply -f deploy.yaml
undep:
	kubectl delete -f deploy.yaml
all: build undep dep

log:
	kubectl logs fluent-sidecar-crd
test:
	kube apply -f deploy-test.yml
untest:
	kube delete -f deploy-test.yml
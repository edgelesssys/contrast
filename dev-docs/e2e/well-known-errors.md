# Well-known CI errors

## 001 - UnauthorizedError pulling container image manifest and config

In `*/generate`, genpolicy rust backtrace:

```
Failed to pull container image manifest and config - error: UnauthorizedError
```
```
=== RUN   TestRegression/serviceaccount/generate
    contrasttest.go:166:
        	Error Trace:	github.com/edgelesssys/contrast/e2e/internal/contrasttest/contrasttest.go:166
        	Error:      	Received unexpected error:
        	            	time=2025-12-06T00:03:21.491Z level=DEBUG msg="running genpolicy" bin=/proc/self/fd/12 args="[--runtime-class-names=contrast-cc --rego-rules-path=/tmp/nix-shell.86HVVp/TestRegression609818586/001/rules.rego --json-settings-path=/tmp/nix-shell.86HVVp/TestRegression609818586/001/settings.json --layers-cache-file-path=/tmp/nix-shell.86HVVp/TestRegression609818586/001/layers-cache.json --yaml-file=/dev/stdin --config-file=/tmp/nix-shell.86HVVp/contrast-generate-extra-2418975012.yml --base64-out]"
        	            	time=2025-12-06T00:03:21.524Z level=DEBUG msg="============================================" position=genpolicy::registry
        	            	time=2025-12-06T00:03:21.524Z level=DEBUG msg="Pulling manifest and config for ghcr.io/edgelesssys/contrast/coordinator@sha256:e552328a72288f098b5e6331d830fcfdfe12188975b679116a7fa2b184e2b31c" position=genpolicy::registry
        	            	time=2025-12-06T00:03:26.751Z level=DEBUG msg=""
        	            	time=2025-12-06T00:03:26.751Z level=ERROR msg="thread 'main' panicked at src/registry.rs:150:17:"
        	            	time=2025-12-06T00:03:26.751Z level=ERROR msg="Failed to pull container image manifest and config - error: UnauthorizedError {"
        	            	time=2025-12-06T00:03:26.751Z level=ERROR msg="    url: \"https://ghcr.io/v2/edgelesssys/contrast/coordinator/manifests/sha256:e552328a72288f098b5e6331d830fcfdfe12188975b679116a7fa2b184e2b31c\","
        	            	time=2025-12-06T00:03:26.751Z level=ERROR msg=}
        	            	time=2025-12-06T00:03:26.751Z level=ERROR msg="stack backtrace:"
        	            	time=2025-12-06T00:03:26.756Z level=ERROR msg="   0: __rustc::rust_begin_unwind"
        	            	time=2025-12-06T00:03:26.756Z level=ERROR msg="   1: core::panicking::panic_fmt"
        	            	time=2025-12-06T00:03:26.757Z level=ERROR msg="   2: genpolicy::pod::Container::init::{{closure}}"
        	            	time=2025-12-06T00:03:26.757Z level=ERROR msg="   3: <genpolicy::stateful_set::StatefulSet as genpolicy::yaml::K8sResource>::init::{{closure}}"
        	            	time=2025-12-06T00:03:26.757Z level=ERROR msg="   4: genpolicy::policy::AgentPolicy::from_files::{{closure}}"
        	            	time=2025-12-06T00:03:26.757Z level=ERROR msg="   5: genpolicy::main::{{closure}}"
        	            	time=2025-12-06T00:03:26.757Z level=ERROR msg="   6: genpolicy::main"
        	            	time=2025-12-06T00:03:26.758Z level=ERROR msg="note: Some details are omitted, run with `RUST_BACKTRACE=full` for a verbose backtrace."
        	            	Error: generate policies: failed to generate policy for "coordinator" in "/tmp/nix-shell.86HVVp/TestRegression609818586/001/resources.yml": running genpolicy: exit status 101

        	            	generate policies: failed to generate policy for "coordinator" in "/tmp/nix-shell.86HVVp/TestRegression609818586/001/resources.yml": running genpolicy: exit status 101
        	Test:       	TestRegression/serviceaccount/generate
=== NAME  TestRegression/serviceaccount
    regression_test.go:111:
        	Error Trace:	github.com/edgelesssys/contrast/e2e/regression/regression_test.go:111
        	Error:      	Should be true
        	Test:       	TestRegression/serviceaccount
        	Messages:   	contrast generate needs to succeed for subsequent tests
```

- https://github.com/edgelesssys/contrast/actions/runs/19978992654/job/57301705959

## 002 - Error reading from server: EOF

```
Error: getting manifests: getting manifests: rpc error: code = Unavailable desc = error reading from server: EOF
```
```
=== RUN   TestImageStore/contrast_verify
time=2025-12-06T01:11:30.669Z level=DEBUG msg="done waiting" namespace=testimagestore-aab0730e-ci condition="PodCondition(one pod matching app.kubernetes.io/name=coordinator is running)"
time=2025-12-06T01:11:30.770Z level=DEBUG msg="done waiting" namespace=testimagestore-aab0730e-ci condition="PodCondition(pod port-forwarder-coordinator is ready)"
time=2025-12-06T01:11:30.871Z level=DEBUG msg="done waiting" namespace=testimagestore-aab0730e-ci condition="PodCondition(pod port-forwarder-coordinator is ready)"
time=2025-12-06T01:11:30.908Z level=INFO msg="forwarded port" attempt=0 namespace=testimagestore-aab0730e-ci pod=port-forwarder-coordinator port=1313 addr=localhost:35989
time=2025-12-06T02:15:07.006Z level=ERROR msg="port-forwarded func failed" attempt=0 namespace=testimagestore-aab0730e-ci pod=port-forwarder-coordinator port=1313 error="running \"verify\": time=2025-12-06T01:11:30.908Z level=DEBUG msg=\"Starting verification\"\ntime=2025-12-06T01:11:30.908Z level=DEBUG msg=\"Using KDS cache dir\" dir=/home/github/.cache/contrast/kds\ntime=2025-12-06T01:11:30.915Z level=DEBUG msg=\"Dialing coordinator\" endpoint=0.0.0.0:35989\ntime=2025-12-06T01:11:30.915Z level=DEBUG msg=\"Getting manifest\"\ntime=2025-12-06T01:11:31.065Z level=INFO msg=\"Validate called\" validator.reference-values=snp-0-MILAN validator.name=snp-0-MILAN validator.report-data=f4080f31395d6831dc8bcc709c43ca8a7f6a6b51d962b7d382c2ed2b0058c7e7a9a81121c05cedd84ec5890026fcccb470535821a635e8b2dede2ee72118049c\ntime=2025-12-06T01:11:31.065Z level=INFO msg=\"Report decoded\" validator.reference-values=snp-0-MILAN validator.report=\"{\\\"version\\\":3, \\\"guestSvn\\\":2, \\\"policy\\\":\\\"196608\\\", \\\"familyId\\\":\\\"AQAAAAAAAAAAAAAAAAAAAA==\\\", \\\"imageId\\\":\\\"AgAAAAAAAAAAAAAAAAAAAA==\\\", \\\"signatureAlgo\\\":1, \\\"currentTcb\\\":\\\"5194620695195156489\\\", \\\"platformInfo\\\":\\\"37\\\", \\\"reportData\\\":\\\"9AgPMTldaDHci8xwnEPKin9qa1HZYrfTgsLtKwBYx+epqBEhwFzt2E7FiQAm/My0cFNYIaY16LLe3i7nIRgEnA==\\\", \\\"measurement\\\":\\\"rS9OlhdhTiYf6vMnwY8uiZsuiGNih9Zf8/8V6vIbIllk7TdqwFf4l6xdeAp6Zb5p\\\", \\\"hostData\\\":\\\"9anaGoDiD67Ss4NvuMe62ZyMNbdFy0wv6FqpQemqugo=\\\", \\\"idKeyDigest\\\":\\\"WxwoccTTFSevQ8PdxN0g2kj6hYID+Ms64gn2AWD/KmhY4JkCKj1F8x9zXx0L03qc\\\", \\\"authorKeyDigest\\\":\\\"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA\\\", \\\"reportId\\\":\\\"cr9USt/t3Gob1ixhsP2vZG+5N6+fJxILWQbFFinMA5A=\\\", \\\"reportIdMa\\\":\\\"//////////////////////////////////////////8=\\\", \\\"reportedTcb\\\":\\\"5194620695195156489\\\", \\\"chipId\\\":\\\"7dWaO1npPOepV6+YoL8Z1d/q5jG5fex9zvtI37K1IjAFdyByRDxMrhyokxxKxeq263g1NCQzxyRO/6Lre4S/lg==\\\", \\\"committedTcb\\\":\\\"5194620695195156489\\\", \\\"currentBuild\\\":39, \\\"currentMinor\\\":55, \\\"currentMajor\\\":1, \\\"committedBuild\\\":39, \\\"committedMinor\\\":55, \\\"committedMajor\\\":1, \\\"launchTcb\\\":\\\"5194620695195156489\\\", \\\"signature\\\":\\\"2fKHPzKaa4Il4g41iBGnOdJnz0pll0y7JjrAkwdRUJDtpB0C+mYBE9eer8+If9VDAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAbk5AS2UDg/w0npYvpyphljtA/qTHvLx7uf8Doz6LNXWrSeJ8TpBvAqgFPLrj9sMIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\\\", \\\"cpuid1eaxFms\\\":10555153}\"\ntime=2025-12-06T01:11:31.065Z level=INFO msg=\"Validate called\" validator.reference-values=snp-1-GENOA validator.name=snp-1-GENOA validator.report-data=f4080f31395d6831dc8bcc709c43ca8a7f6a6b51d962b7d382c2ed2b0058c7e7a9a81121c05cedd84ec5890026fcccb470535821a635e8b2dede2ee72118049c\ntime=2025-12-06T01:11:31.066Z level=INFO msg=\"Report decoded\" validator.reference-values=snp-1-GENOA validator.report=\"{\\\"version\\\":3, \\\"guestSvn\\\":2, \\\"policy\\\":\\\"196608\\\", \\\"familyId\\\":\\\"AQAAAAAAAAAAAAAAAAAAAA==\\\", \\\"imageId\\\":\\\"AgAAAAAAAAAAAAAAAAAAAA==\\\", \\\"signatureAlgo\\\":1, \\\"currentTcb\\\":\\\"5194620695195156489\\\", \\\"platformInfo\\\":\\\"37\\\", \\\"reportData\\\":\\\"9AgPMTldaDHci8xwnEPKin9qa1HZYrfTgsLtKwBYx+epqBEhwFzt2E7FiQAm/My0cFNYIaY16LLe3i7nIRgEnA==\\\", \\\"measurement\\\":\\\"rS9OlhdhTiYf6vMnwY8uiZsuiGNih9Zf8/8V6vIbIllk7TdqwFf4l6xdeAp6Zb5p\\\", \\\"hostData\\\":\\\"9anaGoDiD67Ss4NvuMe62ZyMNbdFy0wv6FqpQemqugo=\\\", \\\"idKeyDigest\\\":\\\"WxwoccTTFSevQ8PdxN0g2kj6hYID+Ms64gn2AWD/KmhY4JkCKj1F8x9zXx0L03qc\\\", \\\"authorKeyDigest\\\":\\\"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA\\\", \\\"reportId\\\":\\\"cr9USt/t3Gob1ixhsP2vZG+5N6+fJxILWQbFFinMA5A=\\\", \\\"reportIdMa\\\":\\\"//////////////////////////////////////////8=\\\", \\\"reportedTcb\\\":\\\"5194620695195156489\\\", \\\"chipId\\\":\\\"7dWaO1npPOepV6+YoL8Z1d/q5jG5fex9zvtI37K1IjAFdyByRDxMrhyokxxKxeq263g1NCQzxyRO/6Lre4S/lg==\\\", \\\"committedTcb\\\":\\\"5194620695195156489\\\", \\\"currentBuild\\\":39, \\\"currentMinor\\\":55, \\\"currentMajor\\\":1, \\\"committedBuild\\\":39, \\\"committedMinor\\\":55, \\\"committedMajor\\\":1, \\\"launchTcb\\\":\\\"5194620695195156489\\\", \\\"signature\\\":\\\"2fKHPzKaa4Il4g41iBGnOdJnz0pll0y7JjrAkwdRUJDtpB0C+mYBE9eer8+If9VDAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAbk5AS2UDg/w0npYvpyphljtA/qTHvLx7uf8Doz6LNXWrSeJ8TpBvAqgFPLrj9sMIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\\\", \\\"cpuid1eaxFms\\\":10555153}\"\ntime=2025-12-06T01:11:31.070Z level=INFO msg=\"Successfully verified report signature\" validator.reference-values=snp-1-GENOA\ntime=2025-12-06T01:11:31.070Z level=INFO msg=\"Validate succeeded\" validator.reference-values=snp-1-GENOA validator.name=snp-1-GENOA validator.report-data=f4080f31395d6831dc8bcc709c43ca8a7f6a6b51d962b7d382c2ed2b0058c7e7a9a81121c05cedd84ec5890026fcccb470535821a635e8b2dede2ee72118049c\nError: getting manifests: getting manifests: rpc error: code = Unavailable desc = error reading from server: EOF\n"
    contrasttest.go:399:
        	Error Trace:	github.com/edgelesssys/contrast/e2e/internal/contrasttest/contrasttest.go:399
        	Error:      	Received unexpected error:
        	            	context deadline exceeded
        	Test:       	TestImageStore/contrast_verify
time=2025-12-06T02:15:07.006Z level=INFO msg="EOF during port-forwarding triggered retry" attempt=0 namespace=testimagestore-aab0730e-ci pod=port-forwarder-coordinator port=1313
time=2025-12-06T02:15:07.013Z level=DEBUG msg="Collecting debug info for pods" namespace=csi-system
```

- https://github.com/edgelesssys/contrast/actions/runs/19978992654/job/57301706570

## 003 - imagepuller: no space left on device

This occurs when the pod VM runs out of disk/memory while the image puller is pulling container images.
Notice that the image puller isn't able to detect all out-of-storage situations currently upfront,
as it only checks the available space based on the size in the manifest before trying to pull.

If the lack of space is detected before pulling, you would see error 004 instead.

This is likely because we misconfigured some memory limit in our Kubernetes resources or because the imagepuller suddenly requires significant more space as expected.

```
verifying and putting layers in store: putting layer to store: unpacking failed (error: exit status 1; output: write /lib/libssl.so.3: no space left on device)
```
```
=== RUN   TestContainerdDigestPinning/contrast_verify
time=2025-12-06T00:29:42.796Z level=DEBUG msg="done waiting" namespace=testcontainerddigestpinning-a1d7884b-ci condition="PodCondition(one pod matching app.kubernetes.io/name=coordinator is running)"
time=2025-12-06T00:29:42.897Z level=DEBUG msg="done waiting" namespace=testcontainerddigestpinning-a1d7884b-ci condition="PodCondition(pod port-forwarder-coordinator is ready)"
time=2025-12-06T00:29:42.997Z level=DEBUG msg="done waiting" namespace=testcontainerddigestpinning-a1d7884b-ci condition="PodCondition(pod port-forwarder-coordinator is ready)"
time=2025-12-06T00:29:43.055Z level=INFO msg="forwarded port" attempt=0 namespace=testcontainerddigestpinning-a1d7884b-ci pod=port-forwarder-coordinator port=1313 addr=localhost:36103
time=2025-12-06T00:29:43.800Z level=DEBUG msg="done waiting" namespace=testcontainerddigestpinning-a1d7884b-ci condition="PodCondition(pod port-forwarder-coordinator is ready)"
time=2025-12-06T00:29:43.837Z level=INFO msg="forwarded port" attempt=0 namespace=testcontainerddigestpinning-a1d7884b-ci pod=port-forwarder-coordinator port=1314 addr=localhost:34737
time=2025-12-06T00:31:04.877Z level=DEBUG msg="context expired while waiting" namespace=testcontainerddigestpinning-a1d7884b-ci condition="PodCondition(1 pods matching app.kubernetes.io/name=containerd-digest-pinning-cc are ready)"
time=2025-12-06T00:31:04.877Z level=DEBUG msg="pod status" namespace=testcontainerddigestpinning-a1d7884b-ci name=containerd-digest-pinning-cc-54fbf655c9-pn7cl status="{\"phase\":\"Running\",\"conditions\":[{\"type\":\"PodReadyToStartContainers\",\"status\":\"True\",\"lastProbeTime\":null,\"lastTransitionTime\":\"2025-12-06T00:29:35Z\"},{\"type\":\"Initialized\",\"status\":\"True\",\"lastProbeTime\":null,\"lastTransitionTime\":\"2025-12-06T00:29:52Z\"},{\"type\":\"Ready\",\"status\":\"False\",\"lastProbeTime\":null,\"lastTransitionTime\":\"2025-12-06T00:29:23Z\",\"reason\":\"ContainersNotReady\",\"message\":\"containers with unready status: [cc-by-digest]\"},{\"type\":\"ContainersReady\",\"status\":\"False\",\"lastProbeTime\":null,\"lastTransitionTime\":\"2025-12-06T00:29:23Z\",\"reason\":\"ContainersNotReady\",\"message\":\"containers with unready status: [cc-by-digest]\"},{\"type\":\"PodScheduled\",\"status\":\"True\",\"lastProbeTime\":null,\"lastTransitionTime\":\"2025-12-06T00:29:23Z\"}],\"hostIP\":\"62.210.145.76\",\"hostIPs\":[{\"ip\":\"62.210.145.76\"}],\"podIP\":\"100.64.0.83\",\"podIPs\":[{\"ip\":\"100.64.0.83\"}],\"startTime\":\"2025-12-06T00:29:23Z\",\"initContainerStatuses\":[{\"name\":\"contrast-initializer\",\"state\":{\"terminated\":{\"exitCode\":0,\"reason\":\"Completed\",\"startedAt\":\"2025-12-06T00:29:35Z\",\"finishedAt\":\"2025-12-06T00:29:47Z\",\"containerID\":\"containerd://af9ee11c37d1bdb960e36e20ad4e13c56d18f0ed07a57bc70170deed203669fd\"}},\"lastState\":{},\"ready\":true,\"restartCount\":0,\"image\":\"sha256:b56424767527d73d73969d49efe0b564c8c1931c7fd9c3ac38c241a168712878\",\"imageID\":\"ghcr.io/edgelesssys/contrast/initializer@sha256:fd259e1f76a1625ce9ea899a064ef8c0dcf2d4a51085332d9ee5490d34a7dae0\",\"containerID\":\"containerd://af9ee11c37d1bdb960e36e20ad4e13c56d18f0ed07a57bc70170deed203669fd\",\"started\":false,\"allocatedResources\":{\"memory\":\"50Mi\"},\"resources\":{\"limits\":{\"memory\":\"50Mi\"},\"requests\":{\"memory\":\"50Mi\"}},\"volumeMounts\":[{\"name\":\"contrast-secrets\",\"mountPath\":\"/contrast\"},{\"name\":\"kube-api-access-kztlb\",\"mountPath\":\"/var/run/secrets/kubernetes.io/serviceaccount\",\"readOnly\":true,\"recursiveReadOnly\":\"Disabled\"}],\"user\":{\"linux\":{\"uid\":0,\"gid\":0,\"supplementalGroups\":[0]}}},{\"name\":\"contrast-debug-shell\",\"state\":{\"running\":{\"startedAt\":\"2025-12-06T00:29:51Z\"}},\"lastState\":{},\"ready\":true,\"restartCount\":0,\"image\":\"sha256:309750e204f800c0e0918c6a7d76101fab33e205e5e9fe8a24a9d25836331f9e\",\"imageID\":\"ghcr.io/edgelesssys/contrast/debugshell@sha256:d2c13f2e3c007ac1c2bb95ba4f5eebf95069dc18595bb964a0b82f6e18f2bf26\",\"containerID\":\"containerd://845d9a69dcfc4aa6c4de6b16caa31024954b3be008a4d4cfe981171d3f6bb5e0\",\"started\":true,\"allocatedResources\":{\"memory\":\"400Mi\"},\"resources\":{\"limits\":{\"memory\":\"400Mi\"},\"requests\":{\"memory\":\"400Mi\"}},\"volumeMounts\":[{\"name\":\"contrast-secrets\",\"mountPath\":\"/contrast\"},{\"name\":\"kube-api-access-kztlb\",\"mountPath\":\"/var/run/secrets/kubernetes.io/serviceaccount\",\"readOnly\":true,\"recursiveReadOnly\":\"Disabled\"}],\"user\":{\"linux\":{\"uid\":0,\"gid\":0,\"supplementalGroups\":[0]}}}],\"containerStatuses\":[{\"name\":\"cc-by-digest\",\"state\":{\"waiting\":{\"reason\":\"CrashLoopBackOff\",\"message\":\"back-off 40s restarting failed container=cc-by-digest pod=containerd-digest-pinning-cc-54fbf655c9-pn7cl_testcontainerddigestpinning-a1d7884b-ci(5508459c-1777-4f69-8c91-4433781a4960)\"}},\"lastState\":{\"terminated\":{\"exitCode\":128,\"reason\":\"StartError\",\"message\":\"failed to create containerd task: failed to create shim task: rpc status: Status { code: UNKNOWN, message: \\\"verifying and putting layers in store: putting layer to store: unpacking failed (error: exit status 1; output: write /lib/libssl.so.3: no space left on device)\\\\nclosing layer reader: %!w(\\u003cnil\\u003e)\\\", details: [], special_fields: SpecialFields { unknown_fields: UnknownFields { fields: None }, cached_size: CachedSize { size: 0 } } }\\n\\nStack backtrace:\\n   0: anyhow::error::\\u003cimpl core::convert::From\\u003cE\\u003e for anyhow::Error\\u003e::from\\n   1: \\u003ckata_agent::storage::image_pull_handler::ImagePullHandler as kata_agent::storage::StorageHandler\\u003e::create_device::{{closure}}\\n   2: kata_agent::storage::add_storages::{{closure}}\\n   3: kata_agent::rpc::AgentService::do_create_container::{{closure}}::{{closure}}\\n   4: \\u003ckata_agent::rpc::AgentService as protocols::agent_ttrpc_async::AgentService\\u003e::create_container::{{closure}}\\n   5: \\u003cprotocols::agent_ttrpc_async::CreateContainerMethod as ttrpc::asynchronous::utils::MethodHandler\\u003e::handler::{{closure}}\\n   6: \\u003ctokio::time::timeout::Timeout\\u003cT\\u003e as core::future::future::Future\\u003e::poll\\n   7: ttrpc::asynchronous::server::HandlerContext::handle_msg::{{closure}}\\n   8: \\u003ccore::future::poll_fn::PollFn\\u003cF\\u003e as core::future::future::Future\\u003e::poll\\n   9: \\u003cttrpc::asynchronous::server::ServerReader as ttrpc::asynchronous::connection::ReaderDelegate\\u003e::handle_msg::{{closure}}::{{closure}}\\n  10: tokio::runtime::task::core::Core\\u003cT,S\\u003e::poll\\n  11: tokio::runtime::task::harness::Harness\\u003cT,S\\u003e::poll\\n  12: tokio::runtime::scheduler::multi_thread::worker::Context::run_task\\n  13: tokio::runtime::scheduler::multi_thread::worker::Context::run\\n  14: tokio::runtime::context::scoped::Scoped\\u003cT\\u003e::set\\n  15: tokio::runtime::context::runtime::enter_runtime\\n  16: tokio::runtime::scheduler::multi_thread::worker::run\\n  17: \\u003ctokio::runtime::blocking::task::BlockingTask\\u003cT\\u003e as core::future::future::Future\\u003e::poll\\n  18: tokio::runtime::task::core::Core\\u003cT,S\\u003e::poll\\n  19: tokio::runtime::task::harness::Harness\\u003cT,S\\u003e::poll\\n  20: tokio::runtime::blocking::pool::Inner::run\\n  21: std::sys::backtrace::__rust_begin_short_backtrace\\n  22: core::ops::function::FnOnce::call_once{{vtable.shim}}\\n  23: std::sys::pal::unix::thread::Thread::new::thread_start\\n  24: start_thread\\n  25: __GI___clone3\",\"startedAt\":\"1970-01-01T00:00:00Z\",\"finishedAt\":\"2025-12-06T00:30:35Z\",\"containerID\":\"containerd://f43727008ba40d624e8e2c7dbac785b2634a56cd927744ba93c16f4314ea6fc5\"}},\"ready\":false,\"restartCount\":3,\"image\":\"ghcr.io/edgelesssys/contrast/containerd-reproducer:1764980939\",\"imageID\":\"ghcr.io/edgelesssys/contrast/containerd-reproducer@sha256:949bb07370c3f34e47e3cadfd34dc1694801a767968c6416c2aa78bd9a32b260\",\"containerID\":\"containerd://f43727008ba40d624e8e2c7dbac785b2634a56cd927744ba93c16f4314ea6fc5\",\"started\":false,\"allocatedResources\":{\"memory\":\"40Mi\"},\"resources\":{\"limits\":{\"memory\":\"40Mi\"},\"requests\":{\"memory\":\"40Mi\"}},\"volumeMounts\":[{\"name\":\"contrast-secrets\",\"mountPath\":\"/contrast\"},{\"name\":\"kube-api-access-kztlb\",\"mountPath\":\"/var/run/secrets/kubernetes.io/serviceaccount\",\"readOnly\":true,\"recursiveReadOnly\":\"Disabled\"}],\"user\":{\"linux\":{\"uid\":0,\"gid\":0,\"supplementalGroups\":[0,1,2,3,4,6,10,11,20,26,27]}}}],\"qosClass\":\"Burstable\"}"
```

- https://github.com/edgelesssys/contrast/actions/runs/19978992654/job/57301706313

## 004 - insufficient storage: pulling xxx would require at least xxx MiB

The imagepuller detected that there is not enough space to pull the image.

This is likely because we misconfigured some memory limit in our Kubernetes resources or because the imagepuller suddenly requires significant more space as expected.

```
insufficient storage: pulling "ghcr.io/edgelesssys/contrast/memdump@sha256:c3330d854496fc92ba89b1399e443b1f84246eeeb244dae1883d5c38d639a838" would require at least 133.9 MiB, but only 56.4 MiB are currently available. Increase the memory limit or image store size
```
```
time=2025-12-06T01:06:56.511Z level=DEBUG msg="pod status" namespace=testmemdump-13bd17e7-ci name=listener-84548765c7-z7vvk status="{\"observedGeneration\":1,\"phase\":\"Running\",\"conditions\":[{\"type\":\"PodReadyToStartContainers\",\"observedGeneration\":1,\"status\":\"True\",\"lastProbeTime\":null,\"lastTransitionTime\":\"2025-12-06T01:04:47Z\"},{\"type\":\"Initialized\",\"observedGeneration\":1,\"status\":\"True\",\"lastProbeTime\":null,\"lastTransitionTime\":\"2025-12-06T01:05:16Z\"},{\"type\":\"Ready\",\"observedGeneration\":1,\"status\":\"False\",\"lastProbeTime\":null,\"lastTransitionTime\":\"2025-12-06T01:04:30Z\",\"reason\":\"ContainersNotReady\",\"message\":\"containers with unready status: [listener]\"},{\"type\":\"ContainersReady\",\"observedGeneration\":1,\"status\":\"False\",\"lastProbeTime\":null,\"lastTransitionTime\":\"2025-12-06T01:04:30Z\",\"reason\":\"ContainersNotReady\",\"message\":\"containers with unready status: [listener]\"},{\"type\":\"PodScheduled\",\"observedGeneration\":1,\"status\":\"True\",\"lastProbeTime\":null,\"lastTransitionTime\":\"2025-12-06T01:04:30Z\"}],\"hostIP\":\"100.93.188.85\",\"hostIPs\":[{\"ip\":\"100.93.188.85\"}],\"podIP\":\"10.42.0.65\",\"podIPs\":[{\"ip\":\"10.42.0.65\"}],\"startTime\":\"2025-12-06T01:04:30Z\",\"initContainerStatuses\":[{\"name\":\"contrast-initializer\",\"state\":{\"terminated\":{\"exitCode\":0,\"reason\":\"Completed\",\"startedAt\":\"2025-12-06T01:04:46Z\",\"finishedAt\":\"2025-12-06T01:05:02Z\",\"containerID\":\"containerd://6b06c768b00fb2e1cf38b4e7153ad17d85269d6f2004cf8bb80f242ce8f29293\"}},\"lastState\":{},\"ready\":true,\"restartCount\":0,\"image\":\"sha256:b56424767527d73d73969d49efe0b564c8c1931c7fd9c3ac38c241a168712878\",\"imageID\":\"ghcr.io/edgelesssys/contrast/initializer@sha256:fd259e1f76a1625ce9ea899a064ef8c0dcf2d4a51085332d9ee5490d34a7dae0\",\"containerID\":\"containerd://6b06c768b00fb2e1cf38b4e7153ad17d85269d6f2004cf8bb80f242ce8f29293\",\"started\":false,\"allocatedResources\":{\"memory\":\"50Mi\"},\"resources\":{\"limits\":{\"memory\":\"50Mi\"},\"requests\":{\"memory\":\"50Mi\"}},\"volumeMounts\":[{\"name\":\"contrast-secrets\",\"mountPath\":\"/contrast\"},{\"name\":\"kube-api-access-64mrt\",\"mountPath\":\"/var/run/secrets/kubernetes.io/serviceaccount\",\"readOnly\":true,\"recursiveReadOnly\":\"Disabled\"}],\"user\":{\"linux\":{\"uid\":0,\"gid\":0,\"supplementalGroups\":[0]}}},{\"name\":\"contrast-debug-shell\",\"state\":{\"running\":{\"startedAt\":\"2025-12-06T01:05:07Z\"}},\"lastState\":{},\"ready\":true,\"restartCount\":0,\"image\":\"sha256:309750e204f800c0e0918c6a7d76101fab33e205e5e9fe8a24a9d25836331f9e\",\"imageID\":\"ghcr.io/edgelesssys/contrast/debugshell@sha256:d2c13f2e3c007ac1c2bb95ba4f5eebf95069dc18595bb964a0b82f6e18f2bf26\",\"containerID\":\"containerd://664406f72ab00262851f87f257a4cc49bf21632fb2b4aadae16d307a135f0a8f\",\"started\":true,\"allocatedResources\":{\"memory\":\"400Mi\"},\"resources\":{\"limits\":{\"memory\":\"400Mi\"},\"requests\":{\"memory\":\"400Mi\"}},\"volumeMounts\":[{\"name\":\"contrast-secrets\",\"mountPath\":\"/contrast\"},{\"name\":\"kube-api-access-64mrt\",\"mountPath\":\"/var/run/secrets/kubernetes.io/serviceaccount\",\"readOnly\":true,\"recursiveReadOnly\":\"Disabled\"}],\"user\":{\"linux\":{\"uid\":0,\"gid\":0,\"supplementalGroups\":[0]}}},{\"name\":\"contrast-service-mesh\",\"state\":{\"running\":{\"startedAt\":\"2025-12-06T01:05:10Z\"}},\"lastState\":{},\"ready\":true,\"restartCount\":0,\"image\":\"sha256:fc925a76948343ec83c15e5a1214a68a88ad10deb995338831ed2224f01e7bcb\",\"imageID\":\"ghcr.io/edgelesssys/contrast/service-mesh-proxy@sha256:c6cc379d1673c245ee5ae8361c35bd5524e8687a2e631ea16ace55942e76efc8\",\"containerID\":\"containerd://9639f557e1aefc10394288f20d937e87fe913c8c1ae75fadc6c663b7171fa9b8\",\"started\":true,\"resources\":{},\"volumeMounts\":[{\"name\":\"contrast-secrets\",\"mountPath\":\"/contrast\"},{\"name\":\"kube-api-access-64mrt\",\"mountPath\":\"/var/run/secrets/kubernetes.io/serviceaccount\",\"readOnly\":true,\"recursiveReadOnly\":\"Disabled\"}],\"user\":{\"linux\":{\"uid\":0,\"gid\":0,\"supplementalGroups\":[0]}}}],\"containerStatuses\":[{\"name\":\"listener\",\"state\":{\"waiting\":{\"reason\":\"CrashLoopBackOff\",\"message\":\"back-off 1m20s restarting failed container=listener pod=listener-84548765c7-z7vvk_testmemdump-13bd17e7-ci(8942d6a8-90e0-4e41-973b-ff10006dc478)\"}},\"lastState\":{\"terminated\":{\"exitCode\":128,\"reason\":\"StartError\",\"message\":\"failed to create containerd task: failed to create shim task: rpc status: Status { code: UNKNOWN, message: \\\"insufficient storage: pulling \\\\\\\"ghcr.io/edgelesssys/contrast/memdump@sha256:c3330d854496fc92ba89b1399e443b1f84246eeeb244dae1883d5c38d639a838\\\\\\\" would require at least 133.9 MiB, but only 56.4 MiB are currently available. Increase the memory limit or image store size\\\", details: [], special_fields: SpecialFields { unknown_fields: UnknownFields { fields: None }, cached_size: CachedSize { size: 0 } } }\\n\\nStack backtrace:\\n   0: anyhow::error::\\u003cimpl core::convert::From\\u003cE\\u003e for anyhow::Error\\u003e::from\\n   1: \\u003ckata_agent::storage::image_pull_handler::ImagePullHandler as kata_agent::storage::StorageHandler\\u003e::create_device::{{closure}}\\n   2: kata_agent::storage::add_storages::{{closure}}\\n   3: kata_agent::rpc::AgentService::do_create_container::{{closure}}::{{closure}}\\n   4: \\u003ckata_agent::rpc::AgentService as protocols::agent_ttrpc_async::AgentService\\u003e::create_container::{{closure}}\\n   5: \\u003cprotocols::agent_ttrpc_async::CreateContainerMethod as ttrpc::asynchronous::utils::MethodHandler\\u003e::handler::{{closure}}\\n   6: \\u003ctokio::time::timeout::Timeout\\u003cT\\u003e as core::future::future::Future\\u003e::poll\\n   7: ttrpc::asynchronous::server::HandlerContext::handle_msg::{{closure}}\\n   8: \\u003ccore::future::poll_fn::PollFn\\u003cF\\u003e as core::future::future::Future\\u003e::poll\\n   9: \\u003cttrpc::asynchronous::server::ServerReader as ttrpc::asynchronous::connection::ReaderDelegate\\u003e::handle_msg::{{closure}}::{{closure}}\\n  10: tokio::runtime::task::core::Core\\u003cT,S\\u003e::poll\\n  11: tokio::runtime::task::harness::Harness\\u003cT,S\\u003e::poll\\n  12: tokio::runtime::scheduler::multi_thread::worker::Context::run_task\\n  13: tokio::runtime::scheduler::multi_thread::worker::Context::run\\n  14: tokio::runtime::context::scoped::Scoped\\u003cT\\u003e::set\\n  15: tokio::runtime::context::runtime::enter_runtime\\n  16: tokio::runtime::scheduler::multi_thread::worker::run\\n  17: \\u003ctokio::runtime::blocking::task::BlockingTask\\u003cT\\u003e as core::future::future::Future\\u003e::poll\\n  18: tokio::runtime::task::core::Core\\u003cT,S\\u003e::poll\\n  19: tokio::runtime::task::harness::Harness\\u003cT,S\\u003e::poll\\n  20: tokio::runtime::blocking::pool::Inner::run\\n  21: std::sys::backtrace::__rust_begin_short_backtrace\\n  22: core::ops::function::FnOnce::call_once{{vtable.shim}}\\n  23: std::sys::pal::unix::thread::Thread::new::thread_start\\n  24: start_thread\\n  25: __GI___clone3\",\"startedAt\":\"1970-01-01T00:00:00Z\",\"finishedAt\":\"2025-12-06T01:06:48Z\",\"containerID\":\"containerd://5f82d36519a85eb160040c6ee609d7c4f0ff7cfb46f57fec8fe2c7e395bc0517\"}},\"ready\":false,\"restartCount\":4,\"image\":\"sha256:9c9510b03af5711066de7ef29eae4d2d77d2589cb297e631295509be6d439595\",\"imageID\":\"ghcr.io/edgelesssys/contrast/memdump@sha256:c3330d854496fc92ba89b1399e443b1f84246eeeb244dae1883d5c38d639a838\",\"containerID\":\"containerd://5f82d36519a85eb160040c6ee609d7c4f0ff7cfb46f57fec8fe2c7e395bc0517\",\"started\":false,\"resources\":{},\"volumeMounts\":[{\"name\":\"contrast-secrets\",\"mountPath\":\"/contrast\"},{\"name\":\"kube-api-access-64mrt\",\"mountPath\":\"/var/run/secrets/kubernetes.io/serviceaccount\",\"readOnly\":true,\"recursiveReadOnly\":\"Disabled\"}],\"user\":{\"linux\":{\"uid\":0,\"gid\":0,\"supplementalGroups\":[0]}}}],\"qosClass\":\"Burstable\"}"
    memdump_test.go:76:
        	Error Trace:	github.com/edgelesssys/contrast/e2e/memdump/memdump_test.go:76
        	Error:      	Received unexpected error:
        	            	context expired while waiting: context deadline exceeded
        	Test:       	TestMemDump/memory_dump_does_not_contain_canary_string
```

## 005 - mcr.microsoft.com: connection reset by peer

Likely a transient network issue while pulling from the Microsoft Container Registry.

Not much we can do about it except retrying the image pull or using another pause image.

```
time=2025-12-06T00:52:31.267Z level=DEBUG msg="Pulling manifest and config for mcr.microsoft.com/oss/kubernetes/pause:3.6" position=genpolicy::registry
time=2025-12-06T00:52:32.530Z level=DEBUG msg=""
time=2025-12-06T00:52:32.530Z level=ERROR msg="thread 'main' panicked at src/registry.rs:150:17:"
time=2025-12-06T00:52:32.530Z level=ERROR msg="Failed to pull container image manifest and config - error: RequestError("
time=2025-12-06T00:52:32.530Z level=ERROR msg="    reqwest::Error {"
time=2025-12-06T00:52:32.530Z level=ERROR msg="        kind: Request,"
time=2025-12-06T00:52:32.530Z level=ERROR msg="        url: \"https://mcr.microsoft.com/v2/oss/kubernetes/pause/manifests/3.6\","
time=2025-12-06T00:52:32.530Z level=ERROR msg="        source: hyper_util::client::legacy::Error("
time=2025-12-06T00:52:32.530Z level=ERROR msg="            SendRequest,"
time=2025-12-06T00:52:32.530Z level=ERROR msg="            hyper::Error("
time=2025-12-06T00:52:32.530Z level=ERROR msg="                Io,"
time=2025-12-06T00:52:32.530Z level=ERROR msg="                Os {"
time=2025-12-06T00:52:32.530Z level=ERROR msg="                    code: 104,"
time=2025-12-06T00:52:32.530Z level=ERROR msg="                    kind: ConnectionReset,"
time=2025-12-06T00:52:32.530Z level=ERROR msg="                    message: \"Connection reset by peer\","
time=2025-12-06T00:52:32.530Z level=ERROR msg="                },"
time=2025-12-06T00:52:32.530Z level=ERROR msg="            ),"
time=2025-12-06T00:52:32.530Z level=ERROR msg="        ),"
time=2025-12-06T00:52:32.530Z level=ERROR msg="    },"
time=2025-12-06T00:52:32.530Z level=ERROR msg=)
time=2025-12-06T00:52:32.530Z level=ERROR msg="stack backtrace:"
time=2025-12-06T00:52:32.536Z level=ERROR msg="   0: __rustc::rust_begin_unwind"
time=2025-12-06T00:52:32.536Z level=ERROR msg="   1: core::panicking::panic_fmt"
time=2025-12-06T00:52:32.536Z level=ERROR msg="   2: genpolicy::pod::Container::init::{{closure}}"
time=2025-12-06T00:52:32.536Z level=ERROR msg="   3: <genpolicy::deployment::Deployment as genpolicy::yaml::K8sResource>::init::{{closure}}"
time=2025-12-06T00:52:32.536Z level=ERROR msg="   4: genpolicy::policy::AgentPolicy::from_files::{{closure}}"
time=2025-12-06T00:52:32.536Z level=ERROR msg="   5: genpolicy::main::{{closure}}"
time=2025-12-06T00:52:32.536Z level=ERROR msg="   6: genpolicy::main"
time=2025-12-06T00:52:32.537Z level=ERROR msg="note: Some details are omitted, run with `RUST_BACKTRACE=full` for a verbose backtrace."
Error: generate policies: failed to generate policy for "openssl-frontend" in "/tmp/nix-shell.K1AGUb/TestDeterminsticPolicyGeneration3638285464/001/resources.yml": running genpolicy: exit status 101
```

- https://github.com/edgelesssys/contrast/actions/runs/19978992654/job/57301706373

## 006 - genpolicy: failed to lookup address information

Likely a transient network issue.

```
error: \"failed to lookup address information: Try again\","
```
```
time=2025-12-08T00:01:09.757Z level=DEBUG msg="running genpolicy" bin=/proc/self/fd/11 args="[--runtime-class-names=contrast-cc --rego-rules-path=/tmp/nix-shell.G7zTj4/TestRegression3597620752/001/rules.rego --json-settings-path=/tmp/nix-shell.G7zTj4/TestRegression3597620752/001/settings.json --layers-cache-file-path=/tmp/nix-shell.G7zTj4/TestRegression3597620752/001/layers-cache.json --yaml-file=/dev/stdin --config-file=/tmp/nix-shell.G7zTj4/contrast-generate-extra-4217488150.yml --base64-out]"
time=2025-12-08T00:01:09.785Z level=DEBUG msg="============================================" position=genpolicy::registry
time=2025-12-08T00:01:09.785Z level=DEBUG msg="Pulling manifest and config for quay.io/fedora/httpd-24-micro@sha256:f8f7d90feb8beace46a9f235e1a215042c7a5d04e1567e11173f7b73ab621a1d" position=genpolicy::registry
time=2025-12-08T00:01:11.049Z level=DEBUG msg="get_users_from_layer: using cache file" position=genpolicy::registry
time=2025-12-08T00:01:11.049Z level=DEBUG msg="Parsing users and groups in image layers" position=genpolicy::registry
time=2025-12-08T00:01:11.049Z level=DEBUG msg="============================================" position=genpolicy::registry
time=2025-12-08T00:01:11.049Z level=DEBUG msg="Pulling manifest and config for mcr.microsoft.com/oss/kubernetes/pause:3.6" position=genpolicy::registry
time=2025-12-08T00:01:11.405Z level=DEBUG msg="get_users_from_layer: using cache file" position=genpolicy::registry
time=2025-12-08T00:01:11.405Z level=DEBUG msg="Parsing users and groups in image layers" position=genpolicy::registry
time=2025-12-08T00:01:11.405Z level=DEBUG msg="============================================" position=genpolicy::registry
time=2025-12-08T00:01:11.405Z level=DEBUG msg="Pulling manifest and config for ghcr.io/edgelesssys/contrast/initializer@sha256:fd259e1f76a1625ce9ea899a064ef8c0dcf2d4a51085332d9ee5490d34a7dae0" position=genpolicy::registry
time=2025-12-08T00:02:12.433Z level=DEBUG msg=""
time=2025-12-08T00:02:12.433Z level=ERROR msg="thread 'main' panicked at src/registry.rs:150:17:"
time=2025-12-08T00:02:12.433Z level=ERROR msg="Failed to pull container image manifest and config - error: RequestError("
time=2025-12-08T00:02:12.433Z level=ERROR msg="    reqwest::Error {"
time=2025-12-08T00:02:12.433Z level=ERROR msg="        kind: Request,"
time=2025-12-08T00:02:12.433Z level=ERROR msg="        url: \"https://ghcr.io/v2/edgelesssys/contrast/initializer/blobs/sha256:b56424767527d73d73969d49efe0b564c8c1931c7fd9c3ac38c241a168712878\","
time=2025-12-08T00:02:12.433Z level=ERROR msg="        source: hyper_util::client::legacy::Error("
time=2025-12-08T00:02:12.433Z level=ERROR msg="            Connect,"
time=2025-12-08T00:02:12.433Z level=ERROR msg="            ConnectError("
time=2025-12-08T00:02:12.433Z level=ERROR msg="                \"dns error\","
time=2025-12-08T00:02:12.433Z level=ERROR msg="                Custom {"
time=2025-12-08T00:02:12.433Z level=ERROR msg="                    kind: Uncategorized,"
time=2025-12-08T00:02:12.433Z level=ERROR msg="                    error: \"failed to lookup address information: Try again\","
time=2025-12-08T00:02:12.433Z level=ERROR msg="                },"
time=2025-12-08T00:02:12.433Z level=ERROR msg="            ),"
time=2025-12-08T00:02:12.433Z level=ERROR msg="        ),"
time=2025-12-08T00:02:12.433Z level=ERROR msg="    },"
time=2025-12-08T00:02:12.433Z level=ERROR msg=)
time=2025-12-08T00:02:12.433Z level=ERROR msg="stack backtrace:"
time=2025-12-08T00:02:12.439Z level=ERROR msg="   0: __rustc::rust_begin_unwind"
time=2025-12-08T00:02:12.439Z level=ERROR msg="   1: core::panicking::panic_fmt"
time=2025-12-08T00:02:12.439Z level=ERROR msg="   2: genpolicy::pod::Container::init::{{closure}}"
time=2025-12-08T00:02:12.439Z level=ERROR msg="   3: <genpolicy::pod::Pod as genpolicy::yaml::K8sResource>::init::{{closure}}"
time=2025-12-08T00:02:12.439Z level=ERROR msg="   4: genpolicy::policy::AgentPolicy::from_files::{{closure}}"
time=2025-12-08T00:02:12.439Z level=ERROR msg="   5: genpolicy::main::{{closure}}"
time=2025-12-08T00:02:12.439Z level=ERROR msg="   6: genpolicy::main"
time=2025-12-08T00:02:12.440Z level=ERROR msg="note: Some details are omitted, run with `RUST_BACKTRACE=full` for a verbose backtrace."
Error: generate policies: failed to generate policy for "privileged-container" in "/tmp/nix-shell.G7zTj4/TestRegression3597620752/001/privileged-container.yml": running genpolicy: exit status 101

generate policies: failed to generate policy for "privileged-container" in "/tmp/nix-shell.G7zTj4/TestRegression3597620752/001/privileged-container.yml": running genpolicy: exit status 101
```

- https://github.com/edgelesssys/contrast/actions/runs/20012080662/job/57383171314

## 007 - AMD KDS down

The AMD Kernel Dynamic Store (KDS) service is down or not reachable.
This happens from time to time as AMD isn't able to operate the service without downtimes.

```
contrasttest.go:399:
        Error Trace:	github.com/edgelesssys/contrast/e2e/internal/contrasttest/contrasttest.go:399
        Error:      	Received unexpected error:
                        running "verify": time=2025-12-07T01:07:17.954Z level=DEBUG msg="Starting verification"
                        time=2025-12-07T01:07:17.955Z level=DEBUG msg="Using KDS cache dir" dir=/home/github/.cache/contrast/kds
                        time=2025-12-07T01:07:17.961Z level=DEBUG msg="Dialing coordinator" endpoint=0.0.0.0:32843
                        time=2025-12-07T01:07:17.961Z level=DEBUG msg="Getting manifest"
                        time=2025-12-07T01:07:37.985Z level=INFO msg="Validate called" validator.reference-values=snp-0-MILAN validator.name=snp-0-MILAN validator.report-data=ae521b19c75c0ddde3ed0db05d73cdfa35f0d04676aeaae9ecd5346e27b6c68b0461982671c2b5c65bc5b5e3a3c9ddeef172aef8a474c8b08d64e92955a05737
                        time=2025-12-07T01:07:37.985Z level=INFO msg="Report decoded" validator.reference-values=snp-0-MILAN validator.report="{\"version\":3,\"guestSvn\":2,\"policy\":\"196608\",\"familyId\":\"AQAAAAAAAAAAAAAAAAAAAA==\",\"imageId\":\"AgAAAAAAAAAAAAAAAAAAAA==\",\"signatureAlgo\":1,\"currentTcb\":\"15355022929519706115\",\"platformInfo\":\"5\",\"reportData\":\"rlIbGcdcDd3j7Q2wXXPN+jXw0EZ2rqrp7NU0bie2xosEYZgmccK1xlvFteOjyd3u8XKu+KR0yLCNZOkpVaBXNw==\",\"measurement\":\"DlCGKalxHC2nA1v/+lPTc9ZQr5SdIff/tOsTwAkhIucPsD8g5/X8Q4Cdw2lvmqwA\",\"hostData\":\"jtWepcutyvcR5k9ad68hhNT+mQkquc/AnHDWtQpwxl4=\",\"idKeyDigest\":\"qQ/INoImU2qAYJVhstImApWkFx7hydGLf/BWiAIv8TRa+Rb6vbg0COQMXiUD2UL1\",\"authorKeyDigest\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"reportId\":\"ixstWSbue4ZtX8ofS8+R0GR3RK9x8Lew/5CFUF01ys8=\",\"reportIdMa\":\"//////////////////////////////////////////8=\",\"reportedTcb\":\"15354741454542995459\",\"chipId\":\"IIs0tchYDqF9UBdSNZe3m/9U4RKEQFreqaB1edX6vJp6A+ppsdIOpYZjf3wJD5TvWI6VOuRNf+g+E0DwxivR/Q==\",\"committedTcb\":\"15355022929519706115\",\"currentBuild\":29,\"currentMinor\":55,\"currentMajor\":1,\"committedBuild\":29,\"committedMinor\":55,\"committedMajor\":1,\"launchTcb\":\"15355022929519706115\",\"signature\":\"plAmscN3lt9RnnvkvpTocvZRM9h+g8IOYbFMEzcdHRY44TiCVAKl6rVhn0De3gotAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAeGCNxEXHr97uKsju9LHapXrPoYdZSl3tXeDuhbzlxBqRgvLhLnsMto0sDYcqrzsUAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\",\"cpuid1eaxFms\":10489617}"
                        time=2025-12-07T01:07:37.985Z level=INFO msg="could not use cached CRL from Coordinator aTLS handshake" validator.reference-values=snp-0-MILAN validator.error="no certificate chain found in attestation data"
                        time=2025-12-07T01:07:37.985Z level=DEBUG msg="Cache hit" kds-getter.url="https://kdsintf.amd.com/vcek/v1/Milan/208b34b5c8580ea17d5017523597b79bff54e11284405adea9a07579d5fabc9a7a03ea69b1d20ea586637f7c090f94ef588e953ae44d7fe83e1340f0c62bd1fd?blSPL=3&teeSPL=0&snpSPL=23&ucodeSPL=213"
                        time=2025-12-07T01:07:37.987Z level=DEBUG msg="Requesting URL" kds-getter.url=https://kdsintf.amd.com/vcek/v1/Milan/crl
                        Error: getting manifests: getting manifests: rpc error: code = Unavailable desc = connection error: desc = "transport: authentication handshake failed: context deadline exceeded"
```

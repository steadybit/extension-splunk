# Steadybit extension-splunk

TODO describe what your extension is doing here from a user perspective.

TODO optionally add your extension to the [Reliability Hub](https://hub.steadybit.com/) by creating
a [pull request](https://github.com/steadybit/reliability-hub-db) and add a link to this README.

## Configuration

| Environment Variable                                         | Helm value                               | Meaning                                                                                                                  | Required | Default |
|--------------------------------------------------------------|------------------------------------------|--------------------------------------------------------------------------------------------------------------------------|----------|---------|
| `STEADYBIT_EXTENSION_ACCESS_TOKEN`                           | `splunk.accessToken`                     | The access token needed to access your splunk observability cloud api.                                                   | Yes      |         |
| `STEADYBIT_EXTENSION_API_BASE_URL`                           | `splunk.apiBaseUrl`                      | The api url for Splunk Observability Cloud, for example `https://app.{realm}.signalfx.com/`                              | Yes      |         |
| `STEADYBIT_EXTENSION_DISCOVERY_ATTRIBUTES_EXCLUDES_DETECTOR` | `discovery.attributes.excludes.detector` | List of Detector Attributes which will be excluded during discovery. Checked by key equality and supporting trailing "*" | No       |         |

The extension supports all environment variables provided by [steadybit/extension-kit](https://github.com/steadybit/extension-kit#environment-variables).

## Installation

### Kubernetes

Detailed information about agent and extension installation in kubernetes can also be found in
our [documentation](https://docs.steadybit.com/install-and-configure/install-agent/install-on-kubernetes).

#### Recommended (via agent helm chart)

All extensions provide a helm chart that is also integrated in the
[helm-chart](https://github.com/steadybit/helm-charts/tree/main/charts/steadybit-agent) of the agent.

You must provide additional values to activate this extension.

```
--set extension-splunk.enabled=true \
```

Additional configuration options can be found in
the [helm-chart](https://github.com/steadybit/extension-splunk/blob/main/charts/steadybit-extension-splunk/values.yaml) of the
extension.

#### Alternative (via own helm chart)

If you need more control, you can install the extension via its
dedicated [helm-chart](https://github.com/steadybit/extension-splunk/blob/main/charts/steadybit-extension-splunk).

```bash
helm repo add steadybit-extension-splunk https://steadybit.github.io/extension-splunk
helm repo update
helm upgrade steadybit-extension-splunk \
    --install \
    --wait \
    --timeout 5m0s \
    --create-namespace \
    --namespace steadybit-agent \
    steadybit-extension-splunk/steadybit-extension-splunk
```

### Linux Package

Please use
our [agent-linux.sh script](https://docs.steadybit.com/install-and-configure/install-agent/install-on-linux-hosts)
to install the extension on your Linux machine. The script will download the latest version of the extension and install
it using the package manager.

After installing, configure the extension by editing `/etc/steadybit/extension-splunk` and then restart the service.

## Extension registration

Make sure that the extension is registered with the agent. In most cases this is done automatically. Please refer to
the [documentation](https://docs.steadybit.com/install-and-configure/install-agent/extension-registration) for more
information about extension registration and how to verify.

## Version and Revision

The version and revision of the extension:
- are printed during the startup of the extension
- are added as a Docker label to the image
- are available via the `version.txt`/`revision.txt` files in the root of the image

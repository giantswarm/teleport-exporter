import logging
from typing import Dict

import pykube
import pytest
from pytest_helm_charts.clusters import Cluster

logger = logging.getLogger(__name__)

namespace_name = "default"
app_name = "teleport-exporter"

timeout: int = 120


@pytest.mark.smoke
def test_api_working(kube_cluster: Cluster) -> None:
    """Very minimalistic example of using the [kube_cluster](pytest_helm_charts.fixtures.kube_cluster)
    fixture to get an instance of [Cluster](pytest_helm_charts.clusters.Cluster) under test
    and access its [kube_client](pytest_helm_charts.clusters.Cluster.kube_client) property
    to get access to Kubernetes API of cluster under test.
    """
    assert kube_cluster.kube_client is not None
    assert len(pykube.Node.objects(kube_cluster.kube_client)) >= 1


@pytest.mark.smoke
def test_cluster_info(
    kube_cluster: Cluster, cluster_type: str, test_extra_info: Dict[str, str]
) -> None:
    """Example shows how you can access additional information about the cluster the tests are running on"""
    logger.info(f"Running on cluster type {cluster_type}")
    key = "external_cluster_type"
    if key in test_extra_info:
        logger.info(f"{key} is {test_extra_info[key]}")
    assert kube_cluster.kube_client is not None
    assert cluster_type != ""


# teleport-exporter dials a real Teleport backend at startup and gates /readyz on
# a successful Ping, so its pod cannot reach Ready in a bare test cluster. The
# smoke test therefore asserts the chart installs and renders the expected
# resources rather than pod readiness.
@pytest.mark.smoke
@pytest.mark.upgrade
@pytest.mark.flaky(reruns=5, reruns_delay=10)
def test_app_resources_created(kube_cluster: Cluster) -> None:
    deployment = pykube.Deployment.objects(
        kube_cluster.kube_client, namespace=namespace_name
    ).get_or_none(name=app_name)
    assert deployment is not None, f"Deployment '{app_name}' was not created"
    assert deployment.obj["spec"]["replicas"] >= 1

    service_account = pykube.ServiceAccount.objects(
        kube_cluster.kube_client, namespace=namespace_name
    ).get_or_none(name=app_name)
    assert service_account is not None, f"ServiceAccount '{app_name}' was not created"

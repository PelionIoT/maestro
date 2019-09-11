## Metrics

<table>
<colgroup>
<col style="width: 40%" />
<col style="width: 12%" />
<col style="width: 46%" />
</colgroup>
<thead>
<tr class="header">
<th style="text-align: left;"><strong>Metric</strong></th>
<th style="text-align: left;"><strong>Type</strong></th>
<th style="text-align: left;"><strong>Description</strong></th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td style="text-align: left;"><code>relays_devicedb_internal_connections</code></td>
<td style="text-align: left;">gauge</td>
<td style="text-align: left;">The number of relays connected to the devicedb node</td>
</tr>
<tr class="even">
<td style="text-align: left;"><code>sites_devicedb_request_durations_seconds</code></td>
<td style="text-align: left;">histogram</td>
<td style="text-align: left;">A histogram of user facing request times</td>
</tr>
<tr class="odd">
<td style="text-align: left;"><code>sites_devicedb_internal_request_counts</code></td>
<td style="text-align: left;">counter</td>
<td style="text-align: left;">Counts requests between nodes inside the cluster. Labels indicate the source and destination of each request</td>
</tr>
<tr class="even">
<td style="text-align: left;"><code>sites_devicedb_internal_request_failures</code></td>
<td style="text-align: left;">counter</td>
<td style="text-align: left;">Counts failed requests between nodes. Can be used to detect network partitions or failed nodes</td>
</tr>
<tr class="odd">
<td style="text-align: left;"><code>devicedb_storage_errors</code></td>
<td style="text-align: left;">counter</td>
<td style="text-align: left;">Count the number of errors returned by the storage driver. Can be used to detect disk storage problems.</td>
</tr>
<tr class="even">
<td style="text-align: left;"><code>devicedb_peer_reachability</code></td>
<td style="text-align: left;">guage</td>
<td style="text-align: left;">A binary guage indicating whether or not the peer is currently reachable from some other peer</td>
</tr>
</tbody>
</table>

|                         | bscgeth  |
|-------------------------|----------|
| **sum**                 | 81.3006s |
| **relative**            | 1.000x   |
| erc20.approval-transfer | 16.7002s |
| erc20.mint              | 17.0332s |
| erc20.transfer          | 12.0918s |
| snailtracer             | 32.542s  |
| ten-thousand-hashes     | 2.9334s  |

|                         | geth    |
|-------------------------|---------|
| **sum**                 | 139.8ms |
| **relative**            | 1.000x  |
| erc20.approval-transfer | 12ms    |
| erc20.mint              | 10.2ms  |
| erc20.transfer          | 15.6ms  |
| snailtracer             | 94ms    |
| ten-thousand-hashes     | 8ms     |

|                         | evmone |
|-------------------------|--------|
| **sum**                 | 50ms   |
| **relative**            | 1.000x |
| erc20.approval-transfer | 8.2ms  |
| erc20.mint              | 3.4ms  |
| erc20.transfer          | 6ms    |
| snailtracer             | 30ms   |
| ten-thousand-hashes     | 2.4ms  |


|                         | revm    |
|-------------------------|---------|
| **sum**                 | 237.4ms |
| **relative**            | 1.000x  |
| erc20.approval-transfer | 30.2ms  |
| erc20.mint              | 14.4ms  |
| erc20.transfer          | 15ms    |
| snailtracer             | 153ms   |
| ten-thousand-hashes     | 24.8ms  |
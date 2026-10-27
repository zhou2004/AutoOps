# KPA 实测证据

- [summary.json](summary.json)：环境、固定镜像、两种指标结果、脚本版本来源及独立清理确认。
- [kpa-concurrency-result.jsonl](kpa-concurrency-result.jsonl)：并发模式成功测试的完整事件输出。
- [kpa-rps-result.jsonl](kpa-rps-result.jsonl)：RPS 模式成功测试的完整事件输出。
- [final-components.json](final-components.json)：测试后的六组件 Ready、0 重启、工作节点调度及节点健康状态。
- [kpa-e2e-rps-executed.py](kpa-e2e-rps-executed.py)：RPS 运行时脚本快照，其 SHA256 已与 master 文件独立比对。
- [kpa-final-result.jsonl](kpa-final-result.jsonl)：最终版额外 30 秒 RPS 实测，包含功能通过、清理确认、最终成功的完整事件。
- [kpa-final-result.log](kpa-final-result.log)：最终轮完整标准输出，含执行端源 SHA256。
- [kpa-e2e-final-executed.py](kpa-e2e-final-executed.py) / [final-run-source.json](final-run-source.json)：最终轮执行前捕获的源码、SHA256、退出码和完成时间。

这两轮原始日志中的 `passed` 表示功能断言通过，发生在清理请求之前。清理结果是随后单独查询确认的：两个测试命名空间均已消失，没有强制移除命名空间 finalizer。

并发轮运行前未记录脚本 SHA256，因此摘要保留为空；没有将 RPS 或最终版本的摘要误记为并发轮版本。并发轮的 HTTP 校验使用普通 Python 下的 assert，清理为顺序删除；RPS 之前改为显式响应校验和 try/finally 清理。

前两轮结束后，主脚本加入 UID 删除前置条件、Service foreground 删除、命名空间删除等待，并把最终 `passed` 移至清理完成之后。随后使用这个最终版本额外完成 30 秒 RPS 实测：896 次负载请求成功、0 失败，实际 `0 → 3 → 0`，两次冷启动均为 2.626 秒。日志顺序为 `functional-passed` → `cleanup-requested` → `cleanup-confirmed` → `passed`，进程退出码为 0。该轮之后未修改脚本；三个测试命名空间又经独立查询确认全部消失。

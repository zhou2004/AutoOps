/**
 * AutoOps 通用工具方法
 */
export default {
    // 展开树形数据方法
    handleTree(data: any[], id?: string, parentId?: string, children?: string) {
        let config = {
            id: id || 'id',
            parentId: parentId || 'parentId',
            childrenList: children || 'children'
        };
        var childrenListMap: Record<string, any[]> = {};
        var nodeIds: Record<string, any> = {};
        var tree: any[] = [];
        for (let d of data) {
            let pid = d[config.parentId];
            if (childrenListMap[pid] == null) {
                childrenListMap[pid] = [];
            }
            nodeIds[d[config.id]] = d;
            childrenListMap[pid].push(d);
        }
        for (let d of data) {
            let pid = d[config.parentId];
            if (nodeIds[pid] == null) {
                tree.push(d);
            }
        }
        for (let t of tree) {
            adaptToChildrenList(t);
        }
        function adaptToChildrenList(o: any) {
            if (childrenListMap[o[config.id]] !== null) {
                o[config.childrenList] = childrenListMap[o[config.id]];
            }
            if (o[config.childrenList]) {
                for (let c of o[config.childrenList]) {
                    adaptToChildrenList(c);
                }
            }
        }
        return tree;
    }
}

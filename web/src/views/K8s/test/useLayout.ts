export function useLayout() {
    const graph = {};
    const layout = (nodes: any, edges: any, direction: string) => {
        return nodes.map((node: any, i: number) => ({
            ...node,
            position: {
                x: direction === 'LR' ? i * 200 : 0,
                y: direction === 'TB' ? i * 150 : 0
            }
        }));
    };
    return { graph, layout };
}

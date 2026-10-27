#!/usr/bin/env python3
import unittest
from render import render


class RenderTests(unittest.TestCase):
    def test_controller_placement_and_bounded_resources(self):
        result = render('worker-01,worker-02', True, 'controllers', 'worker-01')
        self.assertEqual(len(result['items']), 3)
        for item in result['items']:
            spec = item['spec']['template']['spec']
            placement = spec['affinity']['nodeAffinity']['requiredDuringSchedulingIgnoredDuringExecution']['nodeSelectorTerms'][0]['matchExpressions'][0]
            expected = ['worker-01'] if item['metadata']['name'] == 'vpa-admission-controller' else ['worker-01', 'worker-02']
            self.assertEqual(placement['values'], expected)
            self.assertIn(':1.6.0', spec['containers'][0]['image'])
            self.assertEqual(spec['containers'][0]['resources']['limits']['cpu'], '300m')

    def test_rbac_references_stay_within_owned_roles(self):
        items = render('worker-01', False, 'prerequisites')['items']
        roles = {item['metadata']['name'] for item in items if item['kind'] in {'Role', 'ClusterRole'}}
        for item in items:
            if item['kind'] in {'RoleBinding', 'ClusterRoleBinding'}:
                self.assertIn(item['roleRef']['name'], roles)
                self.assertTrue(item['roleRef']['name'].startswith('autoops:vpa:'))
        self.assertEqual(sum(item['kind'] == 'CustomResourceDefinition' for item in items), 2)
        self.assertNotIn('Deployment', {item['kind'] for item in items})

    def test_invalid_placement_is_rejected(self):
        for workers in ['', 'worker-01,$(id)', 'worker-01;id']:
            with self.assertRaises(ValueError):
                render(workers)
        with self.assertRaises(ValueError):
            render('worker-01', admission_worker='master-01')


if __name__ == '__main__':
    unittest.main()

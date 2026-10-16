#!/usr/bin/env python3
import importlib.util
import json
import os
import subprocess
import tempfile
import unittest

REPO = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
HERDR_TAB_SYNC = os.path.join(REPO, "phone-page", "herdr-tab-sync")
MODULE_PATH = os.path.join(REPO, "phone-page", "tabtasks.py")


def load_module():
    spec = importlib.util.spec_from_file_location("tabtasks", MODULE_PATH)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def write(path, content):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as handle:
        handle.write(content)


def tab(tab_id, cwd, workspace="ws", label="tab"):
    return {"tab_id": tab_id, "workspace_id": tab_id.split(":")[0], "workspace_label": workspace, "tab_label": label, "cwd": cwd}


def page(page_id, tab_id, submitted_at=None, title="Page"):
    return {"id": page_id, "title": title, "created_at": "2026-10-10 01:02:03", "herdr_tab_id": tab_id, "submitted_at": submitted_at, "url": f"https://store.example/p/{page_id}"}



class PureFunctionTests(unittest.TestCase):
    def setUp(self):
        self.module = load_module()

    def test_task_for_tab(self):
        pending = {"uuid": "p", "herdrtab": "w1:t1", "status": "pending"}
        waiting = {"uuid": "w", "herdrtab": "w1:t1", "status": "waiting"}
        completed = {"uuid": "c", "herdrtab": "w1:t1", "status": "completed"}
        deleted = {"uuid": "d", "herdrtab": "w1:t1", "status": "deleted"}
        cases = [
            ("pending wins over completed", [completed, pending], "w1:t1", True, "p"),
            ("waiting counts as pending", [waiting], "w1:t1", True, "w"),
            ("completed ignored when pending only", [completed], "w1:t1", True, None),
            ("completed used for attach", [completed], "w1:t1", False, "c"),
            ("deleted never matches", [deleted], "w1:t1", False, None),
            ("other tab never matches", [pending], "w1:t2", False, None),
            ("no herdrtab never matches", [{"uuid": "x", "status": "pending"}], "w1:t1", False, None),
        ]
        for name, tasks, tab_id, pending_only, expected in cases:
            with self.subTest(name):
                task = self.module.task_for_tab(tasks, tab_id, pending_only)
                self.assertEqual(task["uuid"] if task else None, expected)

    def test_project_for(self):
        home = os.path.expanduser("~")
        cases = [
            ("repo folder", "/work/repos/alpha", "alpha"),
            ("trailing slash", "/work/repos/alpha/", "alpha"),
            ("deleted folder", "/work/repos/beta (deleted)", "beta"),
            ("home folder", home, "misc"),
            ("home with slash", home + "/", "misc"),
            ("empty", "", "misc"),
        ]
        for name, cwd, expected in cases:
            with self.subTest(name):
                self.assertEqual(self.module.project_for(cwd), expected)

    def test_delivery_log(self):
        cases = [
            ("empty log", "", (set(), set())),
            ("success", "delivered page=ok tab=w1:t1 at=2026-10-16T01:02:03.000Z\n", ({("ok", "w1:t1")}, set())),
            ("failure", 'page store listener: could not prompt page "bad" tab "w1:t1"\n', (set(), {"bad"})),
            ("ineligible", 'page store listener: no eligible agent for page "bad" tab "w1:t1"\n', (set(), {"bad"})),
            ("unrelated mention", 'other application: page "ok"\n', (set(), set())),
            ("partial success line", "delivered page=ok tab=w1:t1\n", (set(), set())),
            ("page prefix", "delivered page=okay tab=w1:t1 at=2026-10-16T01:02:03.000Z\n", ({("okay", "w1:t1")}, set())),
        ]
        for name, text, expected in cases:
            with self.subTest(name):
                self.assertEqual(self.module.parse_delivery_log(text), expected)

    def test_page_status(self):
        journal = ({("ok", "w1:t1")}, {"bad", "ok"})
        tabs = {"w1:t1"}
        cases = [
            ("success overrides earlier failure", page("ok", "w1:t1", "2026-10-11 00:00:00"), journal, tabs, "answered"),
            ("failure only", page("bad", "w1:t1", "2026-10-11 00:00:00"), journal, tabs, "undelivered"),
            ("no evidence", page("missing", "w1:t1", "2026-10-11 00:00:00"), journal, tabs, "unknown"),
            ("different asking tab", page("ok", "w1:t2", "2026-10-11 00:00:00"), ({("ok", "w1:t1")}, set()), tabs, "unknown"),
            ("open despite log", page("ok", "w1:t1"), journal, tabs, "open"),
            ("tab gone", page("g", "w9:t9"), journal, tabs, "tabgone"),
            ("no tab", page("n", None, "2026-10-11 00:00:00"), journal, tabs, "notab"),
            ("journal unknown", page("ok", "w1:t1", "2026-10-11 00:00:00"), None, tabs, "unknown"),
            ("tabs unknown", page("g", "w9:t9"), journal, None, "unknown"),
        ]
        for name, entry, evidence, tab_ids, expected in cases:
            with self.subTest(name):
                self.assertEqual(self.module.page_status(entry, evidence, tab_ids), expected)
                self.assertEqual(self.module.delivered(entry, evidence) if evidence is not None else False, expected == "answered")

    def test_dry_run_annotation(self):
        cases = [
            ("new annotation", {}, True),
            ("existing annotation", {"annotations": [{"description": "phone-page p answered now"}]}, False),
        ]
        for name, task, expected in cases:
            with self.subTest(name):
                self.assertEqual(self.module.ensure_annotation(task, "phone-page p answered now", True), expected)
                self.assertEqual(task["annotations"], [{"description": "phone-page p answered now"}])


class SyncTests(unittest.TestCase):
    def setUp(self):
        self.module = load_module()
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.platform = self.directory.name
        write(os.path.join(self.platform, "tab-areas.v2.tsv"), "/work/repos/\twork\tWork Eng\n/work/repos/deep/\tpersonal\tMe\n/finance/\tfinance\tOld finance\n/shared/\tshared\tOld shared\n")
        write(os.path.join(self.platform, "tab-overrides.tsv"), "tab\tarea\towner\tnote\nw2:t1\tsidegig\tSide\tmanual\n")
        write(os.path.join(self.platform, "tab-overrides.v2.tsv"), "gamma\tspecial\tfinance\tFin\n")
        self.rules = self.module.Rules(self.platform)

    def test_classification_and_task_fields(self):
        cases = [
            ("standard area uses route", "w1:t1", "/work/repos/alpha", "ws", "tab", "work", "", False),
            ("per-tab override beats prefix", "w2:t1", "/work/repos/beta (deleted)", "ws", "tab", "sidegig", "Side", False),
            ("cos fallback", "w3:t1", "/other/cos-lab", "ws", "tab", "personal", "Chief of Staff", True),
            ("chief-of-staff beats work prefix", "w2H:tV", "/work/repos/garage/chief-of-staff", "ws", "cos", "personal", "Chief of Staff", True),
            ("shared fallback", "w3:t2", "/shared/project", "ws", "tab", "personal", "Chief of Staff", True),
            ("unmatched fallback", "w3:t3", "/nowhere", "ws", "tab", "personal", "Chief of Staff", True),
            ("finance folder", "w3:t4", "/finance/budget", "ws", "tab", "personal", "Finance", False),
            ("finance label", "w3:t5", "/nowhere", "gamma", "special", "personal", "Finance", False),
            ("longest prefix", "w4:t1", "/work/repos/deep/delta", "ws", "tab", "personal", "", False),
            ("no cwd", "w4:t2", "", "ws", "tab", "personal", "Chief of Staff", True),
        ]
        for name, tab_id, cwd, workspace, label, area, owner, triage in cases:
            with self.subTest(name):
                entry = self.module.Tab(tab_id, tab_id.split(":")[0], workspace, label, cwd)
                self.assertEqual(self.rules.classify(entry), (area, owner))
                self.assertEqual(self.rules.triage(entry), triage)
                fields = self.module.tab_task_fields(entry, area, owner, triage)
                self.assertIn(fields["area"], ("work", "sidegig", "personal"))
                self.assertEqual(fields.get("tags", []), ["triage"] if triage else [])
                self.assertEqual(fields["herdrtab"], tab_id)
                self.assertEqual(fields["sourceid"], tab_id)
                self.assertEqual(fields["source"], "herdr")
                self.assertEqual(fields["status"], "pending")

    def test_per_tab_override_beats_label_and_fallback(self):
        write(os.path.join(self.platform, "tab-overrides.tsv"), "tab\tarea\towner\tnote\nw2:t1\tsidegig\tSide\tmanual\nwJ6:t1\tpersonal\tPlatform\tbackup workspace tab, project misc\n")
        rules = self.module.Rules(self.platform)
        entry = self.module.Tab("wJ6:t1", "wJ6", "gamma", "special", os.path.expanduser("~"))
        self.assertEqual(rules.classify(entry), ("personal", "Platform"))
        self.assertFalse(rules.triage(entry))
        self.assertEqual(self.module.tab_task_fields(entry, *rules.classify(entry))["project"], "misc")

    def test_live_listing_keeps_every_tab_and_uses_current_cwd(self):
        cases = [
            ("tab cwd beats agent cwd", {"cwd": "/work/current"}, [{"tab_id": "w1:t1", "cwd": "/work/old"}], "/work/current"),
            ("agent cwd", {}, [{"tab_id": "w1:t1", "cwd": "/work/agent"}], "/work/agent"),
            ("empty agent cwd does not hide usable cwd", {}, [{"tab_id": "w1:t1", "cwd": ""}, {"tab_id": "w1:t1", "cwd": "/work/agent"}], "/work/agent"),
            ("shell tab without agent", {}, [], ""),
        ]
        for name, extra, agents, expected_cwd in cases:
            with self.subTest(name):
                tabs = self.module.tabs_from_listing(
                    [{"workspace_id": "w1", "label": "workspace"}],
                    [{"tab_id": "w1:t1", "workspace_id": "w1", "label": "tab", **extra}],
                    agents,
                )
                self.assertEqual(list(tabs), ["w1:t1"])
                entry = tabs["w1:t1"]
                self.assertEqual((entry.workspace_id, entry.workspace_label, entry.tab_label, entry.cwd), ("w1", "workspace", "tab", expected_cwd))

    def test_explicit_platform_input_uses_newest_file_and_rejects_missing_data(self):
        root = os.path.join(self.platform, "tabs", "2026-10-16")
        for name, generated, label in [("old.json", "2026-10-16T07:00:00+09:00", "old"), ("new.json", "2026-10-16T09:00:00+09:00", "new")]:
            write(os.path.join(root, name), json.dumps({"generated": generated, "tabs": [tab("w1:t1", "/work/repos/alpha", "alpha", label)]}))
        tabs, source = self.module.load_tabs(self.platform, live=False)
        self.assertEqual(tabs["w1:t1"].tab_label, "new")
        self.assertIn("2 files, generated", source)
        cases = [
            ("missing tab id", [{"cwd": "/work"}]),
            ("missing cwd", [{"tab_id": "w1:t1"}]),
            ("empty", []),
        ]
        os.remove(os.path.join(root, "old.json"))
        for name, entries in cases:
            with self.subTest(name):
                write(os.path.join(root, "new.json"), json.dumps({"tabs": entries}))
                with self.assertRaises(self.module.CommandError):
                    self.module.load_tabs(self.platform, live=False)

    def test_input_mode_help(self):
        completed = subprocess.run([HERDR_TAB_SYNC, "--help"], capture_output=True, text=True)
        self.assertEqual(completed.returncode, 0, completed.stderr)
        self.assertIn("--live | --platform", completed.stdout)
        self.assertIn("read live Herdr tabs (default)", completed.stdout)


if __name__ == "__main__":
    unittest.main()

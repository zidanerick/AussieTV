"""Static template contract; not an Unraid GUI installation test."""
import pathlib
import unittest
import xml.etree.ElementTree as ET

ROOT = pathlib.Path(__file__).resolve().parents[1]

class TemplateContract(unittest.TestCase):
    def test_configuration(self):
        root = ET.parse(ROOT / "unraid/AussieTV.xml").getroot()
        self.assertEqual(root.tag, "Container")
        self.assertEqual(root.get("version"), "2")
        self.assertEqual(root.findtext("Network"), "bridge")
        self.assertEqual(root.findtext("Privileged"), "false")
        tag = (ROOT / "RELEASE_VERSION").read_text().strip()
        self.assertEqual(root.findtext("Repository"), "ghcr.io/zidanerick/aussietv:experimental-" + tag)
        self.assertEqual(root.findtext("WebUI"), "")
        configs = root.findall("Config")
        targets = {c.get("Target"): c for c in configs}
        self.assertEqual(len(targets), len(configs))
        self.assertEqual(targets["/data"].get("Type"), "Path")
        self.assertEqual(targets["45781"].get("Mode"), "tcp")
        self.assertEqual(targets["AUSSIETV_WEB_BIND"].text, "0.0.0.0:45781")
        self.assertEqual(targets["AUSSIETV_MODE"].text, "server")
        self.assertEqual(targets["AUSSIETV_WEB_ORIGIN"].get("Required"), "true")
        for key in ("AUSSIETV_WEB_PASSWORD", "AUSSIETV_JELLYFIN_KEY"):
            self.assertEqual(targets[key].get("Mask"), "true")
            self.assertFalse(targets[key].get("Default"))
            self.assertFalse(targets[key].text)
        for key in ("45782", "AUSSIETV_JELLYFIN_BIND"):
            self.assertEqual(targets[key].get("Required"), "false")
            self.assertFalse(targets[key].text)
        self.assertIn("--cap-drop ALL", root.findtext("ExtraParams"))
        self.assertIn("--read-only", root.findtext("ExtraParams"))
        self.assertIn("AUSSIETV_JELLYFIN_BIND= ", (ROOT / "Dockerfile").read_text())

if __name__ == "__main__":
    unittest.main()

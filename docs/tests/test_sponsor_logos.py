import pathlib
import re
import unittest


class SponsorLogoSizingTest(unittest.TestCase):
    def test_sponsor_logos_have_matching_display_height(self):
        index = pathlib.Path(__file__).parents[1] / "index.md"
        sponsor_lines = index.read_text(encoding="utf-8").splitlines()[2:5]
        heights = [re.search(r'height="(\d+)"', line) for line in sponsor_lines]
        self.assertTrue(all(heights), "every sponsor logo needs an explicit display height")
        self.assertEqual({match.group(1) for match in heights}, {"100"})


if __name__ == "__main__":
    unittest.main()

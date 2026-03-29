#!/usr/bin/env python3
"""Generate logo variants from the mroki source logo.

Outputs (all written to docs/assets/brand/):
  - mroki-logo.png              Full logo (original, copied as-is)
  - mroki-logo-icon-light.png    Icon only (no text), cropped as a square
  - mroki-logo-icon-dark.png     Icon only, inverted colors for dark mode
  - favicon-light.ico            Multi-size ICO (16, 32, 48) from icon
  - favicon-light-16x16.png      16x16 favicon PNG from icon
  - favicon-light-32x32.png      32x32 favicon PNG from icon
  - favicon-dark.ico             Multi-size ICO, inverted for dark mode
  - favicon-dark-16x16.png       16x16 favicon PNG, inverted for dark mode
  - favicon-dark-32x32.png       32x32 favicon PNG, inverted for dark mode

Usage:
  python3 scripts/generate_logos.py [--source ~/Downloads/mroki-logo.png]
"""

import argparse
import os
import shutil
from pathlib import Path

from PIL import Image, ImageChops


def find_content_bounds(img, threshold=50):
    """Find the bounding box of non-background content."""
    pixels = img.load()
    w, h = img.size
    bg = pixels[0, 0]

    def is_content(p):
        if isinstance(p, tuple):
            return sum(abs(a - b) for a, b in zip(p[:3], bg[:3])) > threshold
        return abs(p - bg) > threshold

    col_activity = []
    for x in range(w):
        count = sum(1 for y in range(h) if is_content(pixels[x, y]))
        col_activity.append(count)

    row_activity = []
    for y in range(h):
        count = sum(1 for x in range(w) if is_content(pixels[x, y]))
        row_activity.append(count)

    return col_activity, row_activity


def find_icon_region(col_activity, gap_min_width=20):
    """Find the icon region (left of the gap between icon and text)."""
    content_started = False
    gap_start = None
    for x, a in enumerate(col_activity):
        if a > 0:
            if not content_started:
                content_started = True
            elif gap_start is not None:
                gap_width = x - gap_start
                if gap_width >= gap_min_width:
                    return gap_start  # icon ends here
                gap_start = None
        elif content_started and gap_start is None:
            gap_start = x
    return len(col_activity)


def crop_icon(img):
    """Crop the icon portion and make it a padded square."""
    col_activity, _ = find_content_bounds(img)
    icon_end_col = find_icon_region(col_activity)

    # Find actual content bounds within the icon region only
    pixels = img.load()
    w, h = img.size
    bg = pixels[0, 0]
    threshold = 50

    def is_content(p):
        if isinstance(p, tuple):
            return sum(abs(a - b) for a, b in zip(p[:3], bg[:3])) > threshold
        return abs(p - bg) > threshold

    content_cols = [x for x in range(icon_end_col) if col_activity[x] > 0]

    # Only count row activity within the icon columns
    content_rows = []
    for y in range(h):
        for x in range(icon_end_col):
            if is_content(pixels[x, y]):
                content_rows.append(y)
                break

    if not content_cols or not content_rows:
        raise ValueError("Could not detect icon content")

    left = content_cols[0]
    right = content_cols[-1] + 1
    top = content_rows[0]
    bottom = content_rows[-1] + 1

    icon_w = right - left
    icon_h = bottom - top
    size = max(icon_w, icon_h)

    # Add ~10% padding
    padding = int(size * 0.10)
    size += 2 * padding

    # Center the icon in the square
    cx = (left + right) // 2
    cy = (top + bottom) // 2
    half = size // 2

    crop_left = max(0, cx - half)
    crop_top = max(0, cy - half)
    crop_right = min(img.width, cx + half)
    crop_bottom = min(img.height, cy + half)

    cropped = img.crop((crop_left, crop_top, crop_right, crop_bottom))

    # Ensure perfectly square
    final_size = min(cropped.size)
    if cropped.size[0] != cropped.size[1]:
        cropped = cropped.resize((final_size, final_size), Image.LANCZOS)

    return cropped


def remove_background(img, threshold=50):
    """Replace background-colored pixels with transparency."""
    img = img.convert("RGBA")
    pixels = img.load()
    w, h = img.size
    # Sample background color from corners
    bg = pixels[0, 0][:3]

    for y in range(h):
        for x in range(w):
            r, g, b, a = pixels[x, y]
            diff = abs(r - bg[0]) + abs(g - bg[1]) + abs(b - bg[2])
            if diff < threshold:
                pixels[x, y] = (r, g, b, 0)

    return img


def invert_image(img):
    """Invert all colors in an image."""
    if img.mode == "RGBA":
        r, g, b, a = img.split()
        rgb = Image.merge("RGB", (r, g, b))
        inverted = ImageChops.invert(rgb)
        ir, ig, ib = inverted.split()
        return Image.merge("RGBA", (ir, ig, ib, a))
    return ImageChops.invert(img.convert("RGB"))


def generate_favicons(icon_img, out_dir, suffix=""):
    """Generate favicon files from the icon image."""
    s = f"-{suffix}" if suffix else ""
    sizes = {16: f"favicon{s}-16x16.png", 32: f"favicon{s}-32x32.png"}

    for size, filename in sorted(sizes.items()):
        resized = icon_img.resize((size, size), Image.LANCZOS)
        resized.save(out_dir / filename)
        print(f"  ✓ {filename}")

    ico_name = f"favicon{s}.ico"
    icon_img.save(
        out_dir / ico_name,
        format="ICO",
        sizes=[(16, 16), (32, 32), (48, 48)],
    )
    print(f"  ✓ {ico_name} (16, 32, 48)")


def main():
    parser = argparse.ArgumentParser(description="Generate mroki logo variants")
    parser.add_argument(
        "--source",
        default=os.path.expanduser("~/Downloads/mroki-logo.png"),
        help="Path to source logo",
    )
    parser.add_argument(
        "--out",
        default="docs/assets/brand",
        help="Output directory",
    )
    args = parser.parse_args()

    out_dir = Path(args.out)
    out_dir.mkdir(parents=True, exist_ok=True)

    print(f"Source: {args.source}")
    print(f"Output: {out_dir}\n")

    img = Image.open(args.source)

    # 1. Copy full logo as-is (with original background)
    shutil.copy2(args.source, out_dir / "mroki-logo.png")
    print("  ✓ mroki-logo.png (full logo, original)")

    # 2. Crop icon (light) with background removed
    icon = remove_background(crop_icon(img))
    icon.save(out_dir / "mroki-logo-icon-light.png")
    print(f"  ✓ mroki-logo-icon-light.png ({icon.size[0]}x{icon.size[1]})")

    # 3. Light favicons from icon
    generate_favicons(icon, out_dir, suffix="light")

    # 4. Dark mode icon (inverted colors)
    icon_dark = invert_image(icon)
    icon_dark.save(out_dir / "mroki-logo-icon-dark.png")
    print(f"  ✓ mroki-logo-icon-dark.png ({icon_dark.size[0]}x{icon_dark.size[1]})")

    # 5. Dark mode favicons from inverted icon
    generate_favicons(icon_dark, out_dir, suffix="dark")

    print(f"\nDone! All files in {out_dir}/")


if __name__ == "__main__":
    main()

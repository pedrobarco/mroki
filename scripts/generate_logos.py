#!/usr/bin/env python3
"""Generate logo variants from the mroki source logo.

The source logo must be a transparent PNG (RGBA). All variants are
derived from it — no background removal needed.

Outputs (all written to docs/assets/brand/):
  - mroki-logo-banner-light.png  Banner tight-cropped, for light backgrounds
  - mroki-logo-banner-dark.png   Banner tight-cropped, inverted for dark backgrounds
  - mroki-logo-icon-light.png    Icon only (no text), cropped as a square
  - mroki-logo-icon-dark.png     Icon only, inverted colors for dark mode
  - favicon-light.ico            Multi-size ICO (16, 32, 48) from icon
  - favicon-light-16x16.png      16x16 favicon PNG from icon
  - favicon-light-32x32.png      32x32 favicon PNG from icon
  - favicon-dark.ico             Multi-size ICO, inverted for dark mode
  - favicon-dark-16x16.png       16x16 favicon PNG, inverted for dark mode
  - favicon-dark-32x32.png       32x32 favicon PNG, inverted for dark mode

Usage:
  python3 scripts/generate_logos.py [--source docs/assets/brand/mroki-logo.png]
"""

import argparse
from pathlib import Path

from PIL import Image, ImageChops


def is_opaque(pixel, alpha_threshold=10):
    """Check if a pixel is non-transparent."""
    if isinstance(pixel, tuple) and len(pixel) == 4:
        return pixel[3] > alpha_threshold
    return True


def find_content_bounds(img):
    """Find per-column and per-row content activity using alpha channel."""
    pixels = img.load()
    w, h = img.size

    col_activity = []
    for x in range(w):
        count = sum(1 for y in range(h) if is_opaque(pixels[x, y]))
        col_activity.append(count)

    row_activity = []
    for y in range(h):
        count = sum(1 for x in range(w) if is_opaque(pixels[x, y]))
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

    pixels = img.load()
    w, h = img.size

    content_cols = [x for x in range(icon_end_col) if col_activity[x] > 0]

    # Only count row activity within the icon columns
    content_rows = []
    for y in range(h):
        for x in range(icon_end_col):
            if is_opaque(pixels[x, y]):
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


def crop_banner(img, padding_pct=0.10):
    """Crop the full logo tight to its content bounds (wide rectangle)."""
    col_activity, _ = find_content_bounds(img)
    pixels = img.load()
    w, h = img.size

    content_cols = [x for x, a in enumerate(col_activity) if a > 0]
    content_rows = []
    for y in range(h):
        for x in range(w):
            if is_opaque(pixels[x, y]):
                content_rows.append(y)
                break

    if not content_cols or not content_rows:
        raise ValueError("Could not detect content for banner crop")

    left = content_cols[0]
    right = content_cols[-1] + 1
    top = content_rows[0]
    bottom = content_rows[-1] + 1

    content_w = right - left
    content_h = bottom - top
    pad_x = int(content_w * padding_pct)
    pad_y = int(content_h * padding_pct)

    crop_left = max(0, left - pad_x)
    crop_top = max(0, top - pad_y)
    crop_right = min(w, right + pad_x)
    crop_bottom = min(h, bottom + pad_y)

    return img.crop((crop_left, crop_top, crop_right, crop_bottom))


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
        default="docs/assets/brand/mroki-logo.png",
        help="Path to source logo (transparent PNG)",
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

    img = Image.open(args.source).convert("RGBA")

    # 1. Banners — tight crop, light and dark variants
    banner_light = crop_banner(img)
    banner_light.save(out_dir / "mroki-logo-banner-light.png")
    print(f"  ✓ mroki-logo-banner-light.png ({banner_light.size[0]}x{banner_light.size[1]})")

    banner_dark = invert_image(banner_light)
    banner_dark.save(out_dir / "mroki-logo-banner-dark.png")
    print(f"  ✓ mroki-logo-banner-dark.png ({banner_dark.size[0]}x{banner_dark.size[1]})")

    # 2. Crop icon (light)
    icon = crop_icon(img)
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

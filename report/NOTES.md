# Report API - notes

Nonobvious properties of the module. When you notice your intuition is wrong, add a note here.

## Fixed Bugs

### DivBlock background extends beyond content area (2025-01-29)
- **Issue**: Background divs (such as `Band()`) would extend beyond their actual content to cover subsequent content
- **Root cause**: In `DivBlock.Draw()`, the background was drawn using the full `maxHeight` available to the block, but the content only used a portion of that height. This created a "gap" where background showed but no content was drawn.
- **Real cause**: The layout system passed larger `maxHeight` values to blocks during drawing than the content actually needed, causing backgrounds to fill more space than intended.
- **Fix**: Changed the background drawing to first measure all children to calculate the actual content height, then draw background only for that height (plus padding). Background is now drawn after measuring children instead of before.
- **Lesson**: Background areas should be sized based on actual content dimensions, not available space. When content height varies, measure first then draw background accordingly.

### DivBlock large padding causes content to disappear (2025-01-29)
- **Issue**: Footer text (and other div content) would disappear when the div had large padding, even though the background would still render
- **Root cause**: In `DivBlock.Draw()`, the `childrenHeight` was reduced by `b.Padding.Height()`, and then children were drawn with `maxHeight = childrenHeight - currentY`. When padding was large, this could become very small or negative, preventing text from rendering properly.
- **Manifestation**: Reducing padding would "fix" the issue, leading to confusion about positioning vs. rendering problems
- **Fix**: Removed the artificial constraint on children's available height caused by padding. Children now get adequate space to render (at least their measured content height), but are positioned correctly within the padded area. The padding affects positioning and background, but doesn't prevent content rendering.
- **Code change**: Eliminated `childrenHeight -= b.Padding.Height()` and instead calculate `availableHeight` dynamically to ensure children have enough space
- **Lesson**: Padding should affect layout and visual spacing, but should never prevent content from rendering. Always ensure children have adequate space regardless of container padding.

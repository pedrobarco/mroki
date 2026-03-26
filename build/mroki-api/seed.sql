-- Seed a default gate with httpbin live/shadow URLs
INSERT INTO gates (id, live_url, shadow_url)
VALUES (
  'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
  'https://httpbin.org/anything?service=live',
  'https://httpbin.org/anything?service=shadow'
) ON CONFLICT (id) DO NOTHING;


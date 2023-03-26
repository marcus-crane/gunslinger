from django.db import models


class MediaItem(models.Model):
    ANILIST = 'AL'
    PLEX = 'PL'
    STEAM = 'ST'

    EPISODE = 'EP'
    GAMING = 'VG'
    MANGA = 'MN'
    MOVIE = 'MV'
    TRACK = 'TR'

    SOURCE_CHOICES = [
        (ANILIST, 'Anilist'),
        (PLEX, 'Plex'),
        (STEAM, 'Steam'),
    ]

    CATEGORY_CHOICES = [
        (EPISODE, 'Episode'),
        (GAMING, 'Gaming'),
        (MANGA, 'Manga'),
        (MOVIE, 'Movie'),
        (TRACK, 'Track'),
    ]

    created = models.DateTimeField(auto_now_add=True)
    title = models.CharField(max_length=100)
    subtitle = models.CharField(max_length=100, blank=True, default='')
    author = models.CharField(max_length=100)
    category = models.CharField(max_length=2, choices=CATEGORY_CHOICES)
    is_active = models.BooleanField(default=False)
    source = models.CharField(max_length=2, choices=SOURCE_CHOICES)
    image = models.CharField(max_length=150, blank=True, default='')

    def __str__(self):
        return self.title
    
    def __repr__(self):
        return self.__str__()

    class Meta:
        ordering = ['created']
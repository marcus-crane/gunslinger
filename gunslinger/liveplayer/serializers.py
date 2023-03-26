from rest_framework import serializers

from liveplayer.models import MediaItem


class MediaItemSerializer(serializers.ModelSerializer):
    class Meta:
        model = MediaItem
        fields = ['id', 'title', 'subtitle', 'author', 'category', 'is_active', 'source', 'image']